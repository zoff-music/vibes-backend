import { existsSync } from "node:fs";
import { join } from "node:path";
import { safeWrapAsync } from "@vibez/shared";

const isWatch = process.argv.includes("--watch");
const isProd = process.env.NODE_ENV === "production";

// Use Bun.env (Bun automatically loads .env from project root and parent dirs)
const envVars = Bun.env;

// Map variables to VITE_ prefix
const defines: Record<string, string> = {
  "process.env.NODE_ENV": JSON.stringify(isProd ? "production" : "development"),
};

// Map all VITE_ variables and common ones
for (const [key, value] of Object.entries(envVars)) {
  if (key.startsWith("VITE_")) {
    defines[`import.meta.env.${key}`] = JSON.stringify(value);
  } else if (["CAST_APP_ID", "CAST_RECEIVER_URL", "FRONTEND_URL"].includes(key)) {
    defines[`import.meta.env.VITE_${key}`] = JSON.stringify(value);
  }
}

// Default values if missing
if (!defines["import.meta.env.VITE_CAST_APP_ID"]) {
    defines["import.meta.env.VITE_CAST_APP_ID"] = JSON.stringify("1FAF5D9F");
}
if (!defines["import.meta.env.VITE_CAST_RECEIVER_URL"]) {
    defines["import.meta.env.VITE_CAST_RECEIVER_URL"] = JSON.stringify("/casting/receiver/");
}

console.log(`[Build] ${isWatch ? "Watching" : "Building"} with defines:`, Object.keys(defines));

async function runBuild() {
    const result = await Bun.build({
        entrypoints: ["./client.tsx"],
        outdir: "./dist/client",
        minify: isProd,
        define: defines,
    });

    if (!result.success) {
        console.error("Build failed");
        for (const message of result.logs) {
            console.error(message);
        }
        if (!isWatch) process.exit(1);
    } else {
        console.log(`[Build] Success! ${new Date().toLocaleTimeString()}`);
        // Copy public files for Cast app specifically
        const publicDir = join(import.meta.dir, "../public");
        if (existsSync(publicDir)) {
             // Bun doesn't have a simple recursive copy, use shell
             const [copyErr, _] = await safeWrapAsync(
                 Bun.spawn(["cp", "-r", "public/.", "dist/client/"], { cwd: join(import.meta.dir, "..") }).exited
             );
             if (copyErr) {
                 console.error("Failed to copy public files", copyErr);
             }
        }
    }
}

if (isWatch) {
    const watcher = Bun.watch(join(import.meta.dir, ".."), (event, filename) => {
        if (filename?.endsWith(".tsx") || filename?.endsWith(".ts") || filename?.endsWith(".css") || filename?.includes("public/")) {
            console.log(`[Build] Change detected in ${filename}, rebuilding...`);
            runBuild();
        }
    });
}

// Always run initial build
runBuild();
