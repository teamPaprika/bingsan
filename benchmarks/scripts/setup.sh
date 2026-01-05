#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCHMARKS_DIR="$(dirname "$SCRIPT_DIR")"
POLARIS_TOOLS_DIR="$BENCHMARKS_DIR/polaris-tools"

echo "=== Bingsan Benchmark Setup ==="

# Check Java version
check_java() {
    echo "Checking Java installation..."
    if ! command -v java &> /dev/null; then
        echo "ERROR: Java is not installed. Please install Java 17 or later."
        echo "  macOS: brew install openjdk@17"
        echo "  Ubuntu: sudo apt install openjdk-17-jdk"
        exit 1
    fi

    JAVA_VERSION=$(java -version 2>&1 | awk -F '"' '/version/ {print $2}' | cut -d'.' -f1)
    if [ "$JAVA_VERSION" -lt 17 ]; then
        echo "ERROR: Java 17 or later is required. Found version: $JAVA_VERSION"
        exit 1
    fi

    # Java 25+ has compatibility issues with Gradle, prefer Java 17-22
    if [ "$JAVA_VERSION" -ge 25 ]; then
        # Try to find Java 22 or 17
        if /usr/libexec/java_home -v 22 &>/dev/null; then
            export JAVA_HOME=$(/usr/libexec/java_home -v 22)
            echo "Using Java 22 for Gradle compatibility: $JAVA_HOME"
        elif /usr/libexec/java_home -v 17 &>/dev/null; then
            export JAVA_HOME=$(/usr/libexec/java_home -v 17)
            echo "Using Java 17 for Gradle compatibility: $JAVA_HOME"
        else
            echo "WARNING: Java $JAVA_VERSION may have Gradle compatibility issues."
            echo "Consider installing Java 17-22 for best results."
        fi
    fi
    echo "Java version: $(java -version 2>&1 | head -1)"
}

# Clone polaris-tools if not present
clone_polaris_tools() {
    if [ -d "$POLARIS_TOOLS_DIR" ]; then
        echo "polaris-tools already exists, updating..."
        cd "$POLARIS_TOOLS_DIR"
        git fetch origin
        git pull origin main
    else
        echo "Cloning apache/polaris-tools..."
        git clone --depth 1 https://github.com/apache/polaris-tools.git "$POLARIS_TOOLS_DIR"
    fi
}

# Build Gatling simulations
build_simulations() {
    echo "Building Gatling simulations..."
    cd "$POLARIS_TOOLS_DIR/benchmarks"

    # Check if gradle wrapper exists
    if [ -f "./gradlew" ]; then
        ./gradlew classes
    else
        echo "ERROR: Gradle wrapper not found in polaris-tools/benchmarks"
        exit 1
    fi
}

# Create symlink to bingsan config
setup_config() {
    echo "Setting up bingsan configuration..."
    BINGSAN_CONF="$BENCHMARKS_DIR/config/bingsan.conf"
    TARGET_CONF="$POLARIS_TOOLS_DIR/benchmarks/src/gatling/resources/application.conf"

    if [ -f "$BINGSAN_CONF" ]; then
        # Create a combined config that includes defaults and bingsan overrides
        cat > "$TARGET_CONF" << EOF
# Auto-generated config for bingsan benchmarks
include "benchmark-defaults.conf"
include file("$BINGSAN_CONF")
EOF
        echo "Configuration linked: $TARGET_CONF"
    else
        echo "WARNING: $BINGSAN_CONF not found, using default configuration"
    fi
}

# Main
main() {
    check_java
    clone_polaris_tools
    build_simulations
    setup_config

    echo ""
    echo "=== Setup Complete ==="
    echo "Next steps:"
    echo "  1. Start bingsan: make start-bingsan"
    echo "  2. Run benchmarks: make create-dataset && make read-benchmark"
    echo "  3. View results: make report"
}

main "$@"
