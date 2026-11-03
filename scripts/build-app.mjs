import { open, readFile, rename, rm, writeFile } from "node:fs/promises";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const counterPath = path.join(projectRoot, "src-tauri", "build-number.json");
const tauriConfigPath = path.join(projectRoot, "src-tauri", "tauri.conf.json");
const lockPath = path.join(projectRoot, "src-tauri", ".build-number.lock");
const overlayPath = path.join(projectRoot, "src-tauri", `.tauri-build-${process.pid}.json`);

async function writeCounter(buildNumber) {
  const temporary = `${counterPath}.${process.pid}.tmp`;
  await writeFile(temporary, `${JSON.stringify({ buildNumber }, null, 2)}\n`, "utf8");
  await rename(temporary, counterPath);
}

async function runTauri(version) {
  const cli = path.join(projectRoot, "node_modules", "@tauri-apps", "cli", "tauri.js");
  await writeFile(
    overlayPath,
    `${JSON.stringify({ version }, null, 2)}\n`,
    "utf8",
  );
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [cli, "build", "--bundles", "app", "--config", overlayPath], {
      cwd: projectRoot,
      stdio: "inherit",
    });
    child.once("error", reject);
    child.once("exit", (code, signal) => {
      if (code === 0) resolve();
      else reject(new Error(signal ? `Tauri build stopped by ${signal}.` : `Tauri build exited with code ${code}.`));
    });
  });
}

let lock;
let previousBuildNumber;
try {
  lock = await open(lockPath, "wx");
  const counter = JSON.parse(await readFile(counterPath, "utf8"));
  const tauriConfig = JSON.parse(await readFile(tauriConfigPath, "utf8"));
  previousBuildNumber = Number(counter.buildNumber);
  if (!Number.isSafeInteger(previousBuildNumber) || previousBuildNumber < 0) {
    throw new Error("src-tauri/build-number.json must contain a non-negative integer buildNumber.");
  }
  const nextBuildNumber = previousBuildNumber + 1;
  const versionParts = String(tauriConfig.version).split(".");
  if (versionParts.length !== 3 || versionParts.some(part => !/^\d+$/.test(part))) {
    throw new Error("src-tauri/tauri.conf.json version must contain three numeric parts.");
  }
  const version = `${versionParts[0]}.${versionParts[1]}.${nextBuildNumber}`;
  await writeCounter(nextBuildNumber);
  console.log(`Building Plainmark ${version}…`);
  try {
    await runTauri(version);
  } catch (error) {
    await writeCounter(previousBuildNumber);
    throw error;
  }
} finally {
  await rm(overlayPath, { force: true });
  if (lock) {
    await lock.close();
    await rm(lockPath, { force: true });
  }
}
