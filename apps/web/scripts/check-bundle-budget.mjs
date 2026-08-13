import { readFile } from "node:fs/promises";
import { gzipSync } from "node:zlib";
import { resolve } from "node:path";

const maximumCompressedEntryBytes = 350 * 1024;
const distributionDirectory = resolve("dist");
const manifestPath = resolve(distributionDirectory, ".vite", "manifest.json");

const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
const entryChunks = Object.values(manifest).filter((chunk) => chunk.isEntry);

if (entryChunks.length !== 1) {
  throw new Error(
    `Expected one initial JavaScript entry in the Vite manifest, found ${entryChunks.length}.`,
  );
}

const [entryChunk] = entryChunks;
if (typeof entryChunk.file !== "string" || !entryChunk.file.endsWith(".js")) {
  throw new Error("The Vite entry does not reference a JavaScript file.");
}

const entryBytes = await readFile(
  resolve(distributionDirectory, entryChunk.file),
);
const compressedBytes = gzipSync(entryBytes, { level: 9 }).byteLength;

if (compressedBytes >= maximumCompressedEntryBytes) {
  throw new Error(
    `Initial entry ${entryChunk.file} is ${compressedBytes} compressed bytes; the limit is below ${maximumCompressedEntryBytes} bytes.`,
  );
}

console.log(
  `Bundle budget passed: ${entryChunk.file} is ${compressedBytes} compressed bytes (limit: <${maximumCompressedEntryBytes}).`,
);
