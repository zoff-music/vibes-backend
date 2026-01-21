# Troubleshooting Guide

Common issues and solutions for Vibez development.

## Frontend Issues

### TypeScript Errors

#### `Cannot find name 'process'` in API package
**Symptoms**: TypeScript errors about missing `process` global when running `bun run typecheck`

**Solution**: The API package includes a type declaration for Node.js process global in Bun environment. If you see this error:

1. Ensure `@types/node` is installed in the API package:
   ```bash
   cd frontend/packages/api
   bun add -d @types/node
   ```

2. Check that `tsconfig.json` includes `"node"` in the types array:
   ```json
   {
     "compilerOptions": {
       "types": ["vite/client", "node"]
     }
   }
   ```

#### `Property 'accessToken' does not exist on type 'Error'`
**Symptoms**: Type errors in `useProviderToken.ts` about accessing properties on Error objects

**Solution**: This indicates incorrect destructuring of `safeWrapAsync` return values. The correct pattern is:

```typescript
// Correct: [error, data]
const [err, data] = await safeWrapAsync(promise);

// Incorrect: [data, error] 
const [data, err] = await safeWrapAsync(promise);
```

#### Missing `typecheck` script
**Symptoms**: `bun run typecheck` fails with "script not found" for specific packages

**Solution**: Ensure all packages have a `typecheck` script in their `package.json`:

```json
{
  "scripts": {
    "typecheck": "tsc --noEmit"
  }
}
```

### Workspace Issues

#### Package not found errors
**Symptoms**: Import errors for workspace packages like `@vibez/shared`, `@vibez/models`

**Solution**: 
1. Run `bun install` from the frontend root to ensure workspace links are created
2. Check that workspace dependencies use `workspace:*` in package.json:
   ```json
   {
     "dependencies": {
       "@vibez/shared": "workspace:*"
     }
   }
   ```

## Backend Issues

### Database Migration Issues

#### `unable to open database file: no such file or directory`
**Symptoms**: Migrator fails with database path errors

**Solution**: 
1. Ensure the database directory exists:
   ```bash
   mkdir -p data/db
   ```

2. Use correct relative paths:
   - From project root: `./data/db/vibes.db`
   - From migrator directory: `../data/db/vibes.db`

3. For Docker environments, ensure volume mounts are correct in `docker-compose.yml`

#### Migration version conflicts
**Symptoms**: "database is in a dirty state" errors

**Solution**:
1. Check current migration state:
   ```bash
   cd migrator
   go run main.go -db ../data/db/vibes.db
   ```

2. If dirty, manually fix the schema_migrations table or restore from backup

### Docker Issues

#### Invalid migrator flags
**Symptoms**: Docker migrator service fails with unknown flag errors

**Solution**: The migrator only accepts these flags:
- `-db`: Database path (required)
- `-down`: Run down migrations (optional)
- `-steps`: Number of steps (optional)

Remove any `-dir` or other invalid flags from docker-compose.yml commands.

## Development Environment

### Port Conflicts
**Symptoms**: "port already in use" errors when starting services

**Solution**:
1. Check what's using the ports:
   ```bash
   lsof -i :8080  # Backend
   lsof -i :3000  # Platform app
   lsof -i :3001  # Cast app
   ```

2. Kill conflicting processes or change ports in environment variables

### SSL Certificate Issues (Local HTTPS)
**Symptoms**: Browser security warnings when using `make local-dev`

**Solution**:
1. Caddy automatically generates self-signed certificates for localhost
2. Accept the browser warning or add Caddy's root CA to your system trust store
3. Alternative: Use `make dev` for HTTP-only development

### Asset Path Conflicts and Cache Issues
**Symptoms**: Assets not loading, 404 errors for JS/CSS files, or stale cached assets

**Solution**: The project uses separate asset paths to prevent conflicts between apps:

**Asset Path Structure**:
- **Cast app**: `/assets/cast/` - served by cast server (port 3001)
- **Platform app**: `/assets/platform/` - served by platform server (port 3000)
- **Caddy proxy**: Routes asset requests to correct servers based on path

**Cache Busting**:
- Production builds include content hashes in filenames (e.g., `main-abc123.js`)
- Development uses simple filenames for faster rebuilds
- Manifest files track hashed filenames for SSR injection

**Common issues**:

1. **Asset 404 errors**: Check that Caddy is routing `/assets/cast/*` and `/assets/platform/*` correctly
2. **Wrong MIME types**: Ensure servers set `Content-Type: text/css` for CSS files
3. **Stale cache**: Hard refresh (Cmd+Shift+R) or check if hashed filenames are updating
4. **Build failures**: Verify both CSS and JS builds complete successfully

**Debug asset issues**:
```bash
# Check asset routing through Caddy
curl -I https://localhost/assets/cast/index.css
curl -I https://localhost/assets/platform/main.css

# Check direct server responses
curl -I http://localhost:3001/assets/cast/index.css
curl -I http://localhost:3000/assets/platform/main.css

# Verify build outputs
ls frontend/apps/cast/dist/assets/cast/
ls frontend/apps/platform/dist/assets/platform/

# Check manifest files
cat frontend/apps/cast/dist/manifest.json
cat frontend/apps/platform/dist/.vite/manifest.json
```

**Fix asset conflicts**:
1. Ensure each app builds to its own asset directory
2. Verify Caddy routes are in correct order (most specific first)
3. Check that SSR servers read manifest files correctly
4. Confirm build scripts generate proper hashed filenames in production

### SSR (Server-Side Rendering) Issues
**Symptoms**: Hydration mismatches, blank pages, or console errors about SSR

**Solution**: Both platform and cast apps use SSR with different approaches:

**Platform App SSR**:
- Uses Vite with React 19 SSR streaming
- Automatically prefetches room data for `/room/{id}` routes
- Handles redirects for non-existent rooms

**Cast App SSR**:
- Uses custom Bun-based SSR with streaming HTML
- Injects Cast SDK and YouTube API scripts
- Handles static file serving and HMR

**Common SSR issues**:

1. **Hydration mismatches**: Ensure server and client render identical content
2. **Missing initial data**: Check that SSR data injection is working
3. **Asset loading failures**: Verify static file serving is configured correctly
4. **Environment differences**: Ensure NODE_ENV is set correctly

**Debug SSR issues**:
```bash
# Check SSR logs
# Platform app
curl http://localhost:3000/room/test-room

# Cast app  
curl http://localhost:3001

# Verify SSR data injection
curl -s http://localhost:3000 | grep "ssr-data"

# Check for hydration errors in browser console
# Look for "Warning: Text content did not match" or similar
```

**Fix hydration issues**:
- Ensure useEffect hooks don't modify DOM on first render
- Use `suppressHydrationWarning` for dynamic content like timestamps
- Verify Zustand stores are properly initialized with SSR data

### Bun Installation Issues
**Symptoms**: `bun: command not found` or version conflicts

**Solution**:
1. Install Bun via official installer:
   ```bash
   curl -fsSL https://bun.sh/install | bash
   ```

2. Ensure Bun is in your PATH:
   ```bash
   export PATH="$HOME/.bun/bin:$PATH"
   ```

3. Verify installation:
   ```bash
   bun --version
   ```

### Bun.watch Compatibility Issues
**Symptoms**: `TypeError: Bun.watch is not a function` when running cast app in development

**Solution**: This occurs with Bun versions prior to v1.4.0. The cast app build script has been updated to use Node.js `fs.watch` as a fallback:

1. The error typically appears during `make local-dev` when starting the cast app
2. The build script automatically detects if `Bun.watch` is available and falls back to `fs.watch`
3. You should see `[Build] Using fs.watch, watching /path/to/cast` in the logs
4. File watching functionality remains the same - changes to `.tsx`, `.ts`, `.css` files will trigger rebuilds

**Alternative**: Update to Bun v1.4.0+ when available for native `Bun.watch` support.

### Cast Receiver Rendering Issues
**Symptoms**: Cast receiver app doesn't render or shows blank screen at `https://localhost/casting/receiver/`

**Solution**: This typically occurs due to SSR/hydration mismatches, asset path conflicts, or incorrect HTML structure:

1. **Check browser console** for hydration errors or JavaScript errors
2. **Verify asset paths**: Cast assets should load from `/assets/cast/`, platform from `/assets/platform/`
3. **Check network requests**: Ensure CSS and JavaScript assets are loading correctly with proper MIME types
4. **Inspect HTML source**: Should contain proper `<div id="root">` and SSR data script

**Asset Path Issues**:
- Cast app serves assets from `/assets/cast/` with hashed filenames for cache busting
- Platform app serves assets from `/assets/platform/` with hashed filenames
- Caddy proxy routes asset requests to the correct app servers
- Check that `manifest.json` is generated and read correctly for hashed filenames

**Common fixes**:
- Ensure the App component only returns React elements, not HTML tags
- Verify client.tsx hydrates the correct DOM element (`#root`)
- Check that server.tsx properly injects SSR data and assets with correct paths
- Confirm Tailwind CSS is compiled and served from `/assets/cast/index.css`

**Debug steps**:
```bash
# Check if cast server is running and serving assets correctly
curl http://localhost:3001/assets/cast/index.css
curl http://localhost:3001/assets/cast/client.js

# Check CSS compilation
cd frontend/apps/cast
bun run build:css

# Check for TypeScript errors
bun run typecheck

# Verify manifest generation
cat dist/manifest.json
```

## Performance Issues

### Slow TypeScript Checking
**Symptoms**: `bun run typecheck` takes a long time

**Solution**:
1. Use TypeScript project references for better incremental builds
2. Enable `skipLibCheck: true` in tsconfig.json for faster builds
3. Consider using `tsc --build` for workspace-aware compilation

### Slow Frontend Builds
**Symptoms**: Vite dev server or builds are slow

**Solution**:
1. Ensure you're using Bun (faster than npm/yarn)
2. Check for large node_modules - consider `bun install --frozen-lockfile`
3. Use Vite's dependency pre-bundling for better performance

## Testing Issues

### E2E Test Failures
**Symptoms**: Playwright tests fail intermittently

**Solution**:
1. Ensure all services are running and healthy before tests
2. Use proper wait conditions for dynamic content
3. Check test timeouts and increase if needed for slower environments

### API Test Failures
**Symptoms**: Backend tests fail with database errors

**Solution**:
1. Use separate test database: `TEST_DATABASE_PATH=./test.db`
2. Ensure test database is cleaned between runs
3. Run migrations before tests: `make migrate-up`

## Getting Help

If you encounter issues not covered here:

1. Check the logs for specific error messages
2. Verify your environment matches the requirements in README.md
3. Try a clean install: `rm -rf node_modules && bun install`
4. For database issues, check the migrator logs and schema_migrations table
5. Create an issue with full error logs and environment details