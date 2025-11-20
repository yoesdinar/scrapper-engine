# GitHub Token Setup for Private Container Registry

## Step 1: Create a GitHub Personal Access Token (PAT)

### Option A: Fine-grained Personal Access Token (Recommended)

1. Go to https://github.com/settings/tokens?type=beta
2. Click "Generate new token"
3. Configure the token:
   - **Token name**: `Production Server GHCR Access`
   - **Expiration**: `No expiration` (or choose appropriate duration)
   - **Repository access**: Select "Only select repositories" → Choose `scrapper-engine`
   - **Permissions** - Scroll down to find these sections:
     - **Repository permissions** (expand this section):
       - ✅ **Contents**: Read-only (to read repository)
       - ✅ **Metadata**: Read-only (automatically selected)
     - **Account permissions** (scroll down further, separate section at the bottom):
       - ✅ **Packages**: Read (IMPORTANT: This is under Account permissions, not Repository!)
4. Click "Generate token" at the bottom
5. **Copy the token immediately** (you won't see it again!)

**Note:** If you can't find "Packages" under Account permissions in fine-grained tokens, use Option B (Classic Token) instead - it's simpler and works perfectly for this use case.

### Option B: Classic Personal Access Token (Easier - Recommended if Option A is confusing)

1. Go to https://github.com/settings/tokens
2. Click "Generate new token" → "Generate new token (classic)"
3. Configure:
   - **Note**: `Production Server GHCR Access`
   - **Expiration**: `No expiration` (or choose duration)
   - **Select scopes** - Check these boxes:
     - ✅ `read:packages` - Download packages from GitHub Package Registry
     - ✅ `write:packages` - Upload packages to GitHub Package Registry (optional, for pushing)
     - ✅ `repo` - Full control of private repositories (needed for private repos)
4. Click "Generate token" at the bottom
5. **Copy the token immediately** (you won't see it again!)

**This is the simpler option and works perfectly for pulling private container images!**

## Step 2: Add Token to GitHub Secrets

1. Go to your repository: https://github.com/yoesdinar/scrapper-engine
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add:
   - **Name**: `GHCR_TOKEN`
   - **Secret**: Paste your GitHub token from Step 1
5. Click **Add secret**

## Step 3: Verify Secrets Are Set

Make sure you have all these secrets configured:

- ✅ `PROD_HOST` = Your production server IP
- ✅ `PROD_USERNAME` = Your production server username
- ✅ `PROD_PASSWORD` = Your production server password
- ✅ `GHCR_TOKEN` = Your GitHub Personal Access Token

## Step 4: Test the Setup

Commit the updated workflow and trigger deployment:

```bash
git add .github/workflows/deploy-production.yml
git commit -m "Add GitHub token authentication for private images"
git push origin master

# Trigger deployment
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0
git tag v1.0.0
git push origin v1.0.0
```

## Step 5: Manual Testing on Production Server (Optional)

You can test the token manually on your production server:

```bash
# SSH to production
ssh root@YOUR_SERVER_IP

# Login to GHCR with your token
echo "YOUR_TOKEN_HERE" | docker login ghcr.io -u yoesdinar --password-stdin

# Test pulling an image
docker pull ghcr.io/yoesdinar/scrapper-engine-controller:latest

# If successful, you'll see: "Status: Downloaded newer image..."
```

## Troubleshooting

### "unauthorized: unauthenticated"

This means the token is invalid or doesn't have the right permissions.

**Fix:**
1. Verify the token has `read:packages` permission
2. For private repos, ensure token has `repo` scope
3. Generate a new token if needed

### "Error: denied: permission_denied"

The token doesn't have permission to access the package.

**Fix:**
1. Use a Fine-grained token with "Packages: Read" permission
2. Or use Classic token with `read:packages` scope
3. Ensure the token user (yoesdinar) has access to the repository

### "authentication required"

Docker login failed or token expired.

**Fix:**
1. Check if token is expired (go to https://github.com/settings/tokens)
2. Generate a new token with "No expiration"
3. Update the `GHCR_TOKEN` secret in GitHub

### Images still can't be pulled

**Fix:**
1. Make sure images are built and pushed (check Actions tab)
2. Verify image names match in docker-compose:
   ```
   ghcr.io/yoesdinar/scrapper-engine-controller:v1.0.0
   ghcr.io/yoesdinar/scrapper-engine-worker:v1.0.0
   ghcr.io/yoesdinar/scrapper-engine-agent:v1.0.0
   ```
3. Check package visibility (can be private, token will handle auth)

## Security Best Practices

✅ **Use Fine-grained tokens** when possible (more secure)
✅ **Set expiration dates** for tokens (e.g., 1 year)
✅ **Use minimal permissions** (only `read:packages`)
✅ **Rotate tokens regularly** (update GHCR_TOKEN secret when rotated)
✅ **Never commit tokens** to code (always use secrets)
✅ **Revoke unused tokens** at https://github.com/settings/tokens

## Token Rotation

When your token expires or needs rotation:

1. Generate a new token (follow Step 1 above)
2. Update the `GHCR_TOKEN` secret in GitHub
3. Optionally, update on production server:
   ```bash
   ssh root@YOUR_SERVER_IP
   echo "NEW_TOKEN" | docker login ghcr.io -u yoesdinar --password-stdin
   ```

## What the Workflow Does

When you push a tag (e.g., `v1.0.0`), the workflow:

1. ✅ Builds Docker images
2. ✅ Pushes to `ghcr.io/yoesdinar/scrapper-engine-*:v1.0.0`
3. ✅ SSHs to production server
4. ✅ Logs into GHCR using `GHCR_TOKEN`
5. ✅ Pulls new images
6. ✅ Restarts services

All automatically! 🚀
