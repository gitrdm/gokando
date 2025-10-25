# Proton Documentation Setup

Your Go project is now configured with **Proton** for automatic documentation generation.

## Files Created

1. **`.proton/config.yml`** - Proton configuration
   - Configures package discovery, API generation, and examples
   - Customizes GitBook output settings

2. **`.github/workflows/docs.yml`** - GitHub Actions workflow
   - Automatically generates docs when you push to `master` or `main`
   - Deploys generated docs to GitHub Pages

## Local Usage

To generate documentation locally:

```bash
proton generate
```

Output is generated in the `docs/` directory.

## GitHub Pages Setup

1. Go to your repository settings on GitHub
2. Navigate to **Pages** section
3. Set the source to **GitHub Actions**
4. Push changes to trigger the workflow

The workflow will automatically:
- Generate documentation using Proton
- Deploy it to GitHub Pages

## Configuration

To customize documentation generation, edit `.proton/config.yml`:

- **API Generation**: Configure what gets documented
- **GitBook Settings**: Customize the appearance
- **Discovery**: Control which packages are included

For more details, see: https://github.com/kolosys/proton

## Next Steps

1. Commit these files to your repository:
   ```bash
   git add .proton/ .github/ PROTON_SETUP.md
   git commit -m "feat: add automatic documentation generation with Proton"
   ```

2. Enable GitHub Pages in your repository settings

3. Push to trigger the workflow:
   ```bash
   git push
   ```

Your documentation will be available at: `https://gitrdm.github.io/gokando/`
