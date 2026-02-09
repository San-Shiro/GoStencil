# GoStencil — GitHub Pages Deployment Guide

Deploy the WASM webapp so anyone can use GoStencil directly in their browser at `https://yourusername.github.io/GoStencil/`.

---

## Option A: Deploy from `.release/webapp/` (Recommended)

This uses the pre-built WASM binary from the `.release` folder.

### Step 1: Build the Release

```bash
# Already done — the .release/webapp/ folder contains:
#   index.html, style.css, app.js, wasm_exec.js, gostencil.wasm
```

### Step 2: Create a `docs/` folder for GitHub Pages

GitHub Pages can serve from either the repo root or a `docs/` folder. Since you already have a `docs/` folder with documentation, use a **separate branch** instead:

### Step 3: Create and push a `gh-pages` branch

```bash
# From the repo root:
git checkout --orphan gh-pages
git rm -rf .

# Copy webapp files to root of this branch
copy .release\webapp\* .
# Or on Linux/macOS:
# cp .release/webapp/* .

git add index.html style.css app.js wasm_exec.js gostencil.wasm
git commit -m "Deploy GoStencil WASM webapp"
git push origin gh-pages

# Switch back to your main branch
git checkout main
```

### Step 4: Enable GitHub Pages

1. Go to your repository on GitHub: `https://github.com/xob0t/GoStencil`
2. Click **Settings** → **Pages** (left sidebar)
3. Under **Source**, select:
   - **Deploy from a branch**
   - Branch: `gh-pages`
   - Folder: `/ (root)`
4. Click **Save**
5. Wait 1-2 minutes for deployment

### Step 5: Access Your Site

Your app will be live at:
```
https://xob0t.github.io/GoStencil/
```

---

## Option B: Deploy via GitHub Actions (Automated)

This automatically builds and deploys WASM on every push to `main`.

### Step 1: Create the workflow file

Create `.github/workflows/deploy-pages.yml`:

```yaml
name: Deploy WASM to GitHub Pages

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build WASM
        run: |
          GOOS=js GOARCH=wasm go build -ldflags "-s -w" -o clients/wasm/web/gostencil.wasm ./clients/wasm/
          cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" clients/wasm/web/

      - name: Upload Pages artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: clients/wasm/web

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

### Step 2: Enable GitHub Pages (Actions source)

1. Go to **Settings** → **Pages**
2. Under **Source**, select: **GitHub Actions**
3. Push any commit to `main` — the workflow will build and deploy automatically

### Step 3: Access Your Site

Same URL:
```
https://xob0t.github.io/GoStencil/
```

---

## Updating the Deployment

### Option A (Manual Branch)
```bash
# Rebuild
scripts\build_wasm.bat

# Update gh-pages branch
git checkout gh-pages
copy .release\webapp\* . /Y
git add -A
git commit -m "Update WASM build"
git push origin gh-pages
git checkout main
```

### Option B (GitHub Actions)
Just push to `main`. The workflow handles everything automatically.

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Page shows blank/error | Open browser DevTools → Console. Check for WASM loading errors. |
| "Failed to load WASM" | Ensure `gostencil.wasm` is in the same directory as `index.html`. |
| CORS errors | GitHub Pages handles CORS correctly. If testing locally, use `python -m http.server`. Don't open `index.html` directly with `file://`. |
| Large WASM file warning | The .wasm is ~5 MB. GitHub Pages has a 100 MB limit per file, so this is fine. Browser caches it after first load. |
| MIME type error for .wasm | GitHub Pages serves `.wasm` with correct MIME type. If using another host, ensure `application/wasm` MIME type is set. |

---

## Quick Test Locally

Before deploying, test the webapp locally:

```bash
cd .release\webapp
python -m http.server 8080
# Open http://localhost:8080
```

Or if you don't have Python:

```bash
npx -y serve .release/webapp
```
