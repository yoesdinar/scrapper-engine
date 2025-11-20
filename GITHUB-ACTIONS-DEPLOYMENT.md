# GitHub Actions CI/CD Setup

This project uses GitHub Actions to automatically build Docker images and deploy to production when you push a release tag.

## How It Works

1. **Push a release tag** (e.g., `v1.0.0`) to trigger the workflow
2. **GitHub Actions builds** Docker images for controller, worker, and agent
3. **Images are pushed** to GitHub Container Registry (ghcr.io)
4. **Deployment automatically** pulls images and restarts services on production server

## Setup Instructions

### 1. Configure GitHub Repository Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions → New repository secret

Add these secrets:

- **`PROD_HOST`**: Your production server IP address
- **`PROD_USERNAME`**: Your production server username (e.g., `root`)
- **`PROD_PASSWORD`**: Your production server password

### 2. Enable GitHub Container Registry

The workflow uses GitHub Container Registry (ghcr.io) which is automatically available. Images will be published to:

```
ghcr.io/<your-username>/<repo-name>-controller:v1.0.0
ghcr.io/<your-username>/<repo-name>-worker:v1.0.0
ghcr.io/<your-username>/<repo-name>-agent:v1.0.0
```

### 3. Make Images Public (Optional but Recommended)

After first deployment:
1. Go to your GitHub profile → Packages
2. Find your packages (controller, worker, agent)
3. Click on each → Package settings → Change visibility → Public

This allows the production server to pull images without authentication.

## Deployment Process

### Method 1: Create Release via GitHub UI

1. Go to your repository on GitHub
2. Click "Releases" → "Create a new release"
3. Click "Choose a tag" → Type new tag (e.g., `v1.0.0`) → "Create new tag"
4. Fill in release title and description
5. Click "Publish release"
6. GitHub Actions will automatically build and deploy

### Method 2: Create Tag via Command Line

```bash
# Commit your changes
git add .
git commit -m "Your commit message"

# Create and push a version tag
git tag v1.0.0
git push origin v1.0.0
```

### Method 3: Create Tag and Push Together

```bash
# Commit and tag in one go
git add .
git commit -m "Release v1.0.0"
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin main --tags
```

## Monitoring Deployment

### Watch GitHub Actions Progress

1. Go to your repository → Actions tab
2. Click on the running workflow
3. Watch the build and deploy jobs in real-time

### Check Production Server

```bash
# SSH to production
ssh root@103.157.116.91

# Check running containers
cd /root/config-management
docker compose -f docker-compose.production.yml ps

# View logs
docker compose -f docker-compose.production.yml logs -f
```

## Version Tags Format

The workflow triggers on tags matching `v*.*.*` pattern:

- ✅ `v1.0.0` - Works
- ✅ `v1.2.3` - Works
- ✅ `v2.0.0-beta` - Works
- ❌ `1.0.0` - Won't trigger (missing 'v' prefix)
- ❌ `release-1.0` - Won't trigger (wrong format)

## Image Tags Generated

For tag `v1.0.0`, the following tags are created:

- `v1.0.0` - Exact version
- `1.0` - Major.minor version
- `latest` - Latest stable release
- `main-<commit-sha>` - Branch and commit SHA

## Manual Deployment (if needed)

If you need to deploy manually without GitHub Actions:

```bash
# On your local machine
export VERSION=v1.0.0
export REGISTRY=ghcr.io
export IMAGE_PREFIX=<your-username>/<repo-name>

# SSH to production server
ssh root@103.157.116.91

# Pull and deploy
cd /root/config-management
docker compose -f docker-compose.production.yml pull
docker compose -f docker-compose.production.yml up -d
```

## Rollback to Previous Version

```bash
# SSH to production
ssh root@103.157.116.91
cd /root/config-management

# Update .env with previous version
echo "IMAGE_TAG=v1.0.0" > .env
echo "REGISTRY=ghcr.io" >> .env
echo "IMAGE_PREFIX=<your-username>/<repo-name>" >> .env

# Pull and restart
docker compose -f docker-compose.production.yml pull
docker compose -f docker-compose.production.yml up -d
```

## Troubleshooting

### Build Fails

- Check GitHub Actions logs for error messages
- Ensure Dockerfiles are correct
- Verify all dependencies are available

### Deployment Fails

- Verify GitHub secrets are set correctly
- Check SSH access: `ssh root@103.157.116.91`
- Ensure Docker is installed on production server

### Images Not Found

- Check if images are public in GitHub Packages
- Or configure production server to authenticate with ghcr.io:
  ```bash
  echo <GITHUB_TOKEN> | docker login ghcr.io -u <username> --password-stdin
  ```

### Services Not Starting

```bash
# Check logs on production
ssh root@103.157.116.91
cd /root/config-management
docker compose -f docker-compose.production.yml logs
```

## Workflow Files

- `.github/workflows/deploy-production.yml` - Main CI/CD workflow
- `docker-compose.production.yml` - Production deployment configuration

## Benefits of This Approach

✅ **No building on production** - Server only pulls pre-built images
✅ **Faster deployments** - Pre-built images are ready to use
✅ **Version control** - Easy rollback to any previous version
✅ **Automated** - Just push a tag and everything happens automatically
✅ **Consistent builds** - Same images used in testing and production
✅ **Build caching** - GitHub Actions caches layers for faster builds

## Next Steps

1. Set up GitHub secrets
2. Push a test tag: `git tag v0.1.0 && git push origin v0.1.0`
3. Watch the Actions tab to see the build and deployment
4. Verify deployment: `./scripts/test-production.sh`
