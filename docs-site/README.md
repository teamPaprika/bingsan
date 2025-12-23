# Bingsan Documentation

This directory contains the Hugo-based documentation site for Bingsan.

## Prerequisites

- [Hugo](https://gohugo.io/) v0.110.0 or later (extended edition recommended)
- Git (for theme submodule)

### Install Hugo

**macOS (Homebrew):**
```bash
brew install hugo
```

**Linux (Snap):**
```bash
snap install hugo
```

**Windows (Chocolatey):**
```bash
choco install hugo-extended
```

## Local Development

### Clone with Submodules

If cloning fresh, include submodules:
```bash
git clone --recurse-submodules https://github.com/kimuyb/bingsan.git
cd bingsan/docs-site
```

Or initialize submodules after cloning:
```bash
git submodule update --init --recursive
```

### Start Development Server

```bash
hugo server -D
```

The site will be available at `http://localhost:1313/`.

Options:
- `-D` includes draft content
- `--bind 0.0.0.0` to access from other devices
- `-p 8080` to use a different port

### Build for Production

```bash
hugo --minify
```

The static site will be generated in the `public/` directory.

## Project Structure

```
docs-site/
├── archetypes/          # Templates for new content
├── content/
│   ├── _index.md        # Homepage
│   └── docs/            # Documentation pages
│       ├── getting-started/
│       ├── api/
│       ├── configuration/
│       ├── architecture/
│       └── deployment/
├── static/              # Static assets (images, etc.)
├── themes/
│   └── hugo-book/       # Theme (git submodule)
├── hugo.toml            # Site configuration
└── README.md            # This file
```

## Adding Content

### New Documentation Page

```bash
hugo new docs/section-name/page-name.md
```

### Page Front Matter

```yaml
---
title: "Page Title"
weight: 1  # Controls ordering in sidebar
bookCollapseSection: true  # Collapse child pages in sidebar
---
```

### Content Guidelines

1. Use relative links: `{{< relref "/docs/api/tables" >}}`
2. Use markdown alerts for callouts: `> [!WARNING]` or `> [!NOTE]`
3. Include code examples with proper syntax highlighting
4. Keep paragraphs concise

## Deployment

### GitHub Pages

1. Create `.github/workflows/hugo.yml`:

```yaml
name: Deploy Hugo site to Pages

on:
  push:
    branches: ["main"]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

defaults:
  run:
    shell: bash

jobs:
  build:
    runs-on: ubuntu-latest
    env:
      HUGO_VERSION: 0.153.0
    steps:
      - name: Install Hugo CLI
        run: |
          wget -O ${{ runner.temp }}/hugo.deb https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-amd64.deb \
          && sudo dpkg -i ${{ runner.temp }}/hugo.deb
      - name: Checkout
        uses: actions/checkout@v4
        with:
          submodules: recursive
      - name: Setup Pages
        id: pages
        uses: actions/configure-pages@v4
      - name: Build with Hugo
        working-directory: docs-site
        env:
          HUGO_ENVIRONMENT: production
          HUGO_ENV: production
        run: |
          hugo \
            --minify \
            --baseURL "${{ steps.pages.outputs.base_url }}/"
      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: ./docs-site/public

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

2. Enable GitHub Pages in repository settings (Settings > Pages > Source: GitHub Actions)

### Netlify

1. Create `netlify.toml` in the docs-site directory:

```toml
[build]
  command = "hugo --minify"
  publish = "public"

[build.environment]
  HUGO_VERSION = "0.153.0"
  HUGO_ENV = "production"

[context.production.environment]
  HUGO_BASEURL = "https://your-site.netlify.app/"

[context.deploy-preview]
  command = "hugo --minify --buildDrafts --buildFuture -b $DEPLOY_PRIME_URL"

[context.branch-deploy]
  command = "hugo --minify -b $DEPLOY_PRIME_URL"
```

2. Connect your repository to Netlify

### Vercel

1. Create `vercel.json`:

```json
{
  "build": {
    "env": {
      "HUGO_VERSION": "0.153.0"
    }
  },
  "installCommand": "yum install -y wget && wget -O /tmp/hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_Linux-64bit.tar.gz && tar -xzf /tmp/hugo.tar.gz -C /usr/local/bin",
  "buildCommand": "hugo --minify",
  "outputDirectory": "public"
}
```

### Docker

Build a Docker image for the documentation:

```dockerfile
FROM klakegg/hugo:0.153.0-ext-alpine AS builder
WORKDIR /src
COPY . .
RUN hugo --minify

FROM nginx:alpine
COPY --from=builder /src/public /usr/share/nginx/html
EXPOSE 80
```

Build and run:
```bash
docker build -t bingsan-docs .
docker run -p 8080:80 bingsan-docs
```

### Self-Hosted (Nginx)

1. Build the site:
```bash
hugo --minify
```

2. Copy `public/` to your web server:
```bash
rsync -avz public/ user@server:/var/www/docs/
```

3. Configure Nginx:
```nginx
server {
    listen 80;
    server_name docs.bingsan.dev;
    root /var/www/docs;

    location / {
        try_files $uri $uri/ =404;
    }

    error_page 404 /404.html;
}
```

## Theme

This site uses the [Hugo Book](https://github.com/alex-shpak/hugo-book) theme. It's included as a Git submodule.

### Update Theme

```bash
cd themes/hugo-book
git pull origin master
cd ../..
git add themes/hugo-book
git commit -m "Update hugo-book theme"
```

## Troubleshooting

### Theme Not Found

```bash
git submodule update --init --recursive
```

### Build Fails with Git Error

If you see "Failed to read Git log", disable Git info:

```toml
# hugo.toml
enableGitInfo = false
```

### Content Not Appearing

- Check front matter format (YAML between `---`)
- Ensure `draft: true` is removed or build with `-D` flag
- Verify file is in `content/` directory

## License

Documentation is licensed under Apache License 2.0, same as Bingsan itself.
