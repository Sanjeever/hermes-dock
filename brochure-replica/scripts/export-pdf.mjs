import { mkdir, stat } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { chromium } from "playwright";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const brochureDir = resolve(scriptDir, "..");
const outputPath = resolve(brochureDir, "dist", "企智盒宣传册-A4.pdf");
const sourceUrl = pathToFileURL(resolve(brochureDir, "index.html")).href;

await mkdir(dirname(outputPath), { recursive: true });

let browser;

try {
  browser = await chromium.launch({ headless: true });
} catch (error) {
  throw new Error(
    "无法启动 Chromium。请先运行：pnpm --dir brochure-replica exec playwright install chromium chromium-headless-shell",
    { cause: error },
  );
}

try {
  const page = await browser.newPage({ viewport: { width: 1024, height: 1448 } });

  await page.goto(sourceUrl, { waitUntil: "load" });
  await page.evaluate(async () => {
    await document.fonts.ready;
  });
  await page.emulateMedia({ media: "print" });
  await page.pdf({
    path: outputPath,
    format: "A4",
    margin: { top: "0", right: "0", bottom: "0", left: "0" },
    preferCSSPageSize: true,
    printBackground: true,
  });

  const { size } = await stat(outputPath);
  console.log(`已生成 ${outputPath}（${Math.ceil(size / 1024)} KiB）`);
} finally {
  await browser.close();
}
