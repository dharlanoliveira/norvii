import { readdir, readFile } from "node:fs/promises";
import { extname, relative, resolve } from "node:path";

const sourceRoot = resolve("src");
const supportedExtensions = new Set([".ts", ".tsx"]);
const forbiddenPatterns = [
  /research\/demonstration/u,
  /createDemonstrationCatalog/u,
  /European Data Protection Framework/u,
  /brazil-data-protection/u,
  /eu-data-protection/u,
];

const violations = [];

for (const path of await sourceFiles(sourceRoot)) {
  if (path.includes(".test.")) continue;
  const content = await readFile(path, "utf8");
  for (const pattern of forbiddenPatterns) {
    if (pattern.test(content)) {
      violations.push(
        `${relative(sourceRoot, path)} matches ${pattern.source}`,
      );
    }
  }
}

if (violations.length > 0) {
  throw new Error(
    `Production runtime fixture check failed:\n${violations.join("\n")}`,
  );
}

console.log("Production runtime fixture check passed.");

async function sourceFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await sourceFiles(path)));
    } else if (entry.isFile() && supportedExtensions.has(extname(entry.name))) {
      files.push(path);
    }
  }
  return files;
}
