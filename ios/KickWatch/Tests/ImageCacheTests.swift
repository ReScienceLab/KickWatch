import XCTest
import SwiftUI
@testable import KickWatch

final class ImageCacheTests: XCTestCase {

    func testImageForInvalidURLReturnsNil() async {
        let cache = ImageCache(session: .mock())
        MockURLProtocol.requestHandler = { _ in throw URLError(.cannotConnectToHost) }
        let url = URL(string: "https://test.example.com/fail.png")!

        let result = await cache.image(for: url)

        XCTAssertNil(result)
    }

    func testImageForNonImageDataReturnsNil() async {
        let cache = ImageCache(session: .mock())
        MockURLProtocol.requestHandler = { request in
            let response = HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data("not an image".utf8))
        }
        let url = URL(string: "https://test.example.com/text.png")!

        let result = await cache.image(for: url)

        XCTAssertNil(result)
    }

    func testImageIsReturnedForValidPNG() async {
        let cache = ImageCache(session: .mock())
        let pngData = makeSinglePixelPNG()
        MockURLProtocol.requestHandler = { request in
            let response = HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, pngData)
        }
        let url = URL(string: "https://test.example.com/image.png")!

        let result = await cache.image(for: url)

        XCTAssertNotNil(result)
    }

    func testCacheReturnsSameImageOnSecondCall() async {
        let cache = ImageCache(session: .mock())
        let pngData = makeSinglePixelPNG()
        var callCount = 0
        MockURLProtocol.requestHandler = { request in
            callCount += 1
            let response = HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, pngData)
        }
        let url = URL(string: "https://test.example.com/cached.png")!

        _ = await cache.image(for: url)
        _ = await cache.image(for: url)

        XCTAssertEqual(callCount, 1, "Second call should use cache, not make a network request")
    }

    // Minimal 1×1 red PNG
    private func makeSinglePixelPNG() -> Data {
        let renderer = UIGraphicsImageRenderer(size: CGSize(width: 1, height: 1))
        let image = renderer.image { ctx in
            UIColor.red.setFill()
            ctx.fill(CGRect(x: 0, y: 0, width: 1, height: 1))
        }
        return image.pngData()!
    }
}
