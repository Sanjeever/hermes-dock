import AppKit
import Foundation
import PDFKit

let scriptURL = URL(fileURLWithPath: #filePath)
let brochureDir = scriptURL.deletingLastPathComponent().deletingLastPathComponent()
let inputURL = brochureDir.appendingPathComponent("dist/企智盒宣传册-A4.pdf")
let outputURL = brochureDir.appendingPathComponent("dist/企智盒宣传册-A4.png")
let dpi: CGFloat = 300
let pointsPerInch: CGFloat = 72

guard let document = PDFDocument(url: inputURL), let page = document.page(at: 0) else {
    fatalError("无法读取 PDF：\(inputURL.path)")
}

let pageBounds = page.bounds(for: .mediaBox)
let scale = dpi / pointsPerInch
let imageSize = NSSize(width: pageBounds.width * scale, height: pageBounds.height * scale)
let image = NSImage(size: imageSize)

image.lockFocus()
guard let context = NSGraphicsContext.current?.cgContext else {
    fatalError("无法创建 PNG 绘图上下文")
}

context.setFillColor(NSColor.white.cgColor)
context.fill(CGRect(origin: .zero, size: imageSize))
context.saveGState()
context.scaleBy(x: scale, y: scale)
page.draw(with: .mediaBox, to: context)
context.restoreGState()
image.unlockFocus()

guard let tiffData = image.tiffRepresentation,
      let bitmap = NSBitmapImageRep(data: tiffData),
      let pngData = bitmap.representation(using: .png, properties: [:]) else {
    fatalError("无法编码 PNG")
}

try pngData.write(to: outputURL)
print("已生成 \(outputURL.path)（\(Int(imageSize.width)) × \(Int(imageSize.height))，\(Int(dpi)) DPI）")
