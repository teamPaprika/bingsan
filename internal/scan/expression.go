package scan

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/kimuyb/bingsan/internal/avro"
)

// ExprType represents the type of filter expression.
type ExprType string

// Expression types matching the Iceberg spec.
const (
	ExprTrue          ExprType = "true"
	ExprFalse         ExprType = "false"
	ExprAnd           ExprType = "and"
	ExprOr            ExprType = "or"
	ExprNot           ExprType = "not"
	ExprEq            ExprType = "eq"
	ExprNotEq         ExprType = "not-eq"
	ExprLt            ExprType = "lt"
	ExprLtEq          ExprType = "lt-eq"
	ExprGt            ExprType = "gt"
	ExprGtEq          ExprType = "gt-eq"
	ExprIsNull        ExprType = "is-null"
	ExprNotNull       ExprType = "not-null"
	ExprIsNaN         ExprType = "is-nan"
	ExprNotNaN        ExprType = "not-nan"
	ExprIn            ExprType = "in"
	ExprNotIn         ExprType = "not-in"
	ExprStartsWith    ExprType = "starts-with"
	ExprNotStartsWith ExprType = "not-starts-with"
)

// Expression represents a parsed filter expression.
type Expression struct {
	Type   ExprType
	Term   *Term         // For predicate expressions
	Value  any           // For literal comparison
	Values []any         // For set expressions (IN, NOT IN)
	Left   *Expression   // For AND/OR
	Right  *Expression   // For AND/OR
	Child  *Expression   // For NOT
}

// Term represents a column reference.
type Term struct {
	ColumnID   int
	ColumnName string
}

// ParseExpression parses a filter expression from the API request format.
func ParseExpression(data map[string]any) (*Expression, error) {
	if data == nil {
		return nil, nil
	}

	exprType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("expression missing 'type' field")
	}

	expr := &Expression{Type: ExprType(exprType)}

	switch ExprType(exprType) {
	case ExprTrue, ExprFalse:
		// No additional fields needed

	case ExprAnd, ExprOr:
		left, ok := data["left"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s expression missing 'left' field", exprType)
		}
		right, ok := data["right"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s expression missing 'right' field", exprType)
		}

		var err error
		expr.Left, err = ParseExpression(left)
		if err != nil {
			return nil, fmt.Errorf("parsing left expression: %w", err)
		}
		expr.Right, err = ParseExpression(right)
		if err != nil {
			return nil, fmt.Errorf("parsing right expression: %w", err)
		}

	case ExprNot:
		child, ok := data["child"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("not expression missing 'child' field")
		}
		var err error
		expr.Child, err = ParseExpression(child)
		if err != nil {
			return nil, fmt.Errorf("parsing child expression: %w", err)
		}

	case ExprEq, ExprNotEq, ExprLt, ExprLtEq, ExprGt, ExprGtEq, ExprStartsWith, ExprNotStartsWith:
		term, err := parseTerm(data["term"])
		if err != nil {
			return nil, fmt.Errorf("parsing term: %w", err)
		}
		expr.Term = term
		expr.Value = data["value"]

	case ExprIsNull, ExprNotNull, ExprIsNaN, ExprNotNaN:
		term, err := parseTerm(data["term"])
		if err != nil {
			return nil, fmt.Errorf("parsing term: %w", err)
		}
		expr.Term = term

	case ExprIn, ExprNotIn:
		term, err := parseTerm(data["term"])
		if err != nil {
			return nil, fmt.Errorf("parsing term: %w", err)
		}
		expr.Term = term
		values, ok := data["values"].([]any)
		if !ok {
			return nil, fmt.Errorf("%s expression missing 'values' field", exprType)
		}
		expr.Values = values

	default:
		return nil, fmt.Errorf("unknown expression type: %s", exprType)
	}

	return expr, nil
}

// parseTerm parses a term (column reference) from expression data.
func parseTerm(data any) (*Term, error) {
	switch v := data.(type) {
	case map[string]any:
		// Reference term with column ID or name
		term := &Term{}
		if id, ok := v["term-id"].(float64); ok {
			term.ColumnID = int(id)
		}
		if name, ok := v["name"].(string); ok {
			term.ColumnName = name
		}
		if ref, ok := v["reference"].(string); ok {
			term.ColumnName = ref
		}
		return term, nil
	case string:
		// Simple column name reference
		return &Term{ColumnName: v}, nil
	default:
		return nil, fmt.Errorf("invalid term type: %T", data)
	}
}

// Evaluator evaluates filter expressions against file column bounds.
type Evaluator struct {
	schema        []SchemaField
	columnByID    map[int]SchemaField
	columnByName  map[string]SchemaField
	caseSensitive bool
}

// SchemaField contains schema field metadata.
type SchemaField struct {
	ID       int
	Name     string
	Type     string
	Required bool
}

// NewEvaluator creates an evaluator from table schema.
func NewEvaluator(schema []map[string]any, caseSensitive bool) *Evaluator {
	e := &Evaluator{
		columnByID:    make(map[int]SchemaField),
		columnByName:  make(map[string]SchemaField),
		caseSensitive: caseSensitive,
	}

	for _, field := range schema {
		sf := SchemaField{}
		if id, ok := field["id"].(float64); ok {
			sf.ID = int(id)
		}
		if name, ok := field["name"].(string); ok {
			sf.Name = name
		}
		if typ, ok := field["type"].(string); ok {
			sf.Type = typ
		}
		if req, ok := field["required"].(bool); ok {
			sf.Required = req
		}

		e.schema = append(e.schema, sf)
		e.columnByID[sf.ID] = sf
		if caseSensitive {
			e.columnByName[sf.Name] = sf
		} else {
			e.columnByName[strings.ToLower(sf.Name)] = sf
		}
	}

	return e
}

// ShouldIncludeFile determines if a file may contain matching rows.
// Returns true if the file should be included in the scan.
// Uses column min/max bounds to prune files that definitely don't match.
func (e *Evaluator) ShouldIncludeFile(expr *Expression, file *avro.DataFile) bool {
	if expr == nil {
		return true
	}

	switch expr.Type {
	case ExprTrue:
		return true

	case ExprFalse:
		return false

	case ExprAnd:
		return e.ShouldIncludeFile(expr.Left, file) && e.ShouldIncludeFile(expr.Right, file)

	case ExprOr:
		return e.ShouldIncludeFile(expr.Left, file) || e.ShouldIncludeFile(expr.Right, file)

	case ExprNot:
		// For NOT, we can't prune conservatively - must include the file
		// unless we can prove the child is always true
		return true

	case ExprIsNull:
		return e.evaluateIsNull(expr.Term, file)

	case ExprNotNull:
		return e.evaluateNotNull(expr.Term, file)

	case ExprEq:
		return e.evaluateComparison(expr.Term, expr.Value, file, compareEq)

	case ExprNotEq:
		// Can't prune on not-equal
		return true

	case ExprLt:
		return e.evaluateComparison(expr.Term, expr.Value, file, compareLt)

	case ExprLtEq:
		return e.evaluateComparison(expr.Term, expr.Value, file, compareLtEq)

	case ExprGt:
		return e.evaluateComparison(expr.Term, expr.Value, file, compareGt)

	case ExprGtEq:
		return e.evaluateComparison(expr.Term, expr.Value, file, compareGtEq)

	case ExprIn:
		return e.evaluateIn(expr.Term, expr.Values, file)

	case ExprNotIn:
		// Can't prune on not-in
		return true

	case ExprStartsWith:
		return e.evaluateStartsWith(expr.Term, expr.Value, file)

	case ExprNotStartsWith:
		return true

	default:
		// Unknown expression type - include file to be safe
		return true
	}
}

// getColumnID resolves a term to a column ID.
func (e *Evaluator) getColumnID(term *Term) (int, bool) {
	if term.ColumnID > 0 {
		return term.ColumnID, true
	}
	name := term.ColumnName
	if !e.caseSensitive {
		name = strings.ToLower(name)
	}
	if field, ok := e.columnByName[name]; ok {
		return field.ID, true
	}
	return 0, false
}

// evaluateIsNull checks if a file may contain null values for a column.
func (e *Evaluator) evaluateIsNull(term *Term, file *avro.DataFile) bool {
	colID, ok := e.getColumnID(term)
	if !ok {
		return true // Unknown column, include file
	}

	// Check null value counts if available
	if file.NullValueCounts != nil {
		if nullCount, ok := file.NullValueCounts[int32(colID)]; ok {
			return nullCount > 0
		}
	}

	// No stats available, include file
	return true
}

// evaluateNotNull checks if a file may contain non-null values for a column.
func (e *Evaluator) evaluateNotNull(term *Term, file *avro.DataFile) bool {
	colID, ok := e.getColumnID(term)
	if !ok {
		return true
	}

	// Check if all values are null
	if file.NullValueCounts != nil && file.ValueCounts != nil {
		nullCount, hasNull := file.NullValueCounts[int32(colID)]
		totalCount, hasTotal := file.ValueCounts[int32(colID)]
		if hasNull && hasTotal && nullCount >= totalCount {
			return false // All values are null
		}
	}

	return true
}

// CompareFunc is a comparison function type.
type CompareFunc func(lower, upper, value []byte) bool

// evaluateComparison evaluates a comparison predicate against file bounds.
func (e *Evaluator) evaluateComparison(term *Term, value any, file *avro.DataFile, cmp CompareFunc) bool {
	colID, ok := e.getColumnID(term)
	if !ok {
		return true
	}

	lower := file.LowerBounds[int32(colID)]
	upper := file.UpperBounds[int32(colID)]

	if lower == nil && upper == nil {
		return true // No bounds, include file
	}

	// Convert value to bytes for comparison
	valueBytes := valueToBytes(value)
	if valueBytes == nil {
		return true
	}

	return cmp(lower, upper, valueBytes)
}

// evaluateIn checks if any value in the set overlaps with file bounds.
func (e *Evaluator) evaluateIn(term *Term, values []any, file *avro.DataFile) bool {
	colID, ok := e.getColumnID(term)
	if !ok {
		return true
	}

	lower := file.LowerBounds[int32(colID)]
	upper := file.UpperBounds[int32(colID)]

	if lower == nil && upper == nil {
		return true
	}

	// Check if any value falls within [lower, upper]
	for _, v := range values {
		valueBytes := valueToBytes(v)
		if valueBytes == nil {
			continue
		}
		// Value is in range if: lower <= value <= upper
		if (lower == nil || bytes.Compare(lower, valueBytes) <= 0) &&
			(upper == nil || bytes.Compare(valueBytes, upper) <= 0) {
			return true
		}
	}

	return false
}

// evaluateStartsWith checks if the file may contain values starting with prefix.
func (e *Evaluator) evaluateStartsWith(term *Term, value any, file *avro.DataFile) bool {
	colID, ok := e.getColumnID(term)
	if !ok {
		return true
	}

	prefix, ok := value.(string)
	if !ok {
		return true
	}

	lower := file.LowerBounds[int32(colID)]
	upper := file.UpperBounds[int32(colID)]

	if lower == nil && upper == nil {
		return true
	}

	// For prefix matching:
	// - Lower bound must start with prefix OR be less than prefix
	// - Upper bound must start with prefix OR be greater than prefix
	prefixBytes := []byte(prefix)

	// If upper bound is less than prefix, no match
	if upper != nil && bytes.Compare(upper, prefixBytes) < 0 {
		return false
	}

	// If lower bound is greater than prefix + max suffix, no match
	// We approximate this by checking if lower > prefix repeated max times
	if lower != nil {
		// Create a value that's greater than any string starting with prefix
		prefixMax := make([]byte, len(prefixBytes)+1)
		copy(prefixMax, prefixBytes)
		prefixMax[len(prefixBytes)] = 0xFF

		if bytes.Compare(lower, prefixMax) > 0 {
			return false
		}
	}

	return true
}

// Comparison functions

func compareEq(lower, upper, value []byte) bool {
	// For equality, value must be in [lower, upper]
	if lower != nil && bytes.Compare(value, lower) < 0 {
		return false
	}
	if upper != nil && bytes.Compare(value, upper) > 0 {
		return false
	}
	return true
}

func compareLt(lower, upper, value []byte) bool {
	// For value < X, file must have lower < X
	if lower != nil && bytes.Compare(lower, value) >= 0 {
		return false
	}
	return true
}

func compareLtEq(lower, upper, value []byte) bool {
	// For value <= X, file must have lower <= X
	if lower != nil && bytes.Compare(lower, value) > 0 {
		return false
	}
	return true
}

func compareGt(lower, upper, value []byte) bool {
	// For value > X, file must have upper > X
	if upper != nil && bytes.Compare(upper, value) <= 0 {
		return false
	}
	return true
}

func compareGtEq(lower, upper, value []byte) bool {
	// For value >= X, file must have upper >= X
	if upper != nil && bytes.Compare(upper, value) < 0 {
		return false
	}
	return true
}

// valueToBytes converts a value to bytes for comparison.
// This is a simplified implementation - production would need schema-aware encoding.
func valueToBytes(v any) []byte {
	switch val := v.(type) {
	case string:
		return []byte(val)
	case []byte:
		return val
	case int:
		return int64ToBytes(int64(val))
	case int32:
		return int64ToBytes(int64(val))
	case int64:
		return int64ToBytes(val)
	case float64:
		if val == float64(int64(val)) {
			return int64ToBytes(int64(val))
		}
		return float64ToBytes(val)
	case float32:
		return float64ToBytes(float64(val))
	case bool:
		if val {
			return []byte{1}
		}
		return []byte{0}
	default:
		return nil
	}
}

func int64ToBytes(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

func float64ToBytes(v float64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(v))
	return buf
}
