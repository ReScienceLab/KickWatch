import XCTest
@testable import KickWatch

final class APIClientTests: XCTestCase {
    private var client: APIClient!

    override func setUp() async throws {
        client = APIClient(baseURL: "https://test.example.com", urlSession: .mock())
    }

    override func tearDown() async throws {
        MockURLProtocol.requestHandler = nil
        client = nil
    }

    // MARK: - fetchCampaigns URL building

    func testFetchCampaignsURLIncludesSort() async throws {
        var capturedRequest: URLRequest?
        MockURLProtocol.requestHandler = { request in
            capturedRequest = request
            return self.makeOKResponse(request: request, body: """
            {"campaigns":[],"next_cursor":null,"total":0}
            """)
        }

        _ = try await client.fetchCampaigns(sort: "newest", categoryID: nil, cursor: nil)

        let items = queryItems(from: capturedRequest)
        XCTAssertTrue(items.contains(URLQueryItem(name: "sort", value: "newest")))
    }

    func testFetchCampaignsURLIncludesCategoryID() async throws {
        var capturedRequest: URLRequest?
        MockURLProtocol.requestHandler = { request in
            capturedRequest = request
            return self.makeOKResponse(request: request, body: """
            {"campaigns":[],"next_cursor":null,"total":0}
            """)
        }

        _ = try await client.fetchCampaigns(sort: "trending", categoryID: "16", cursor: nil)

        let items = queryItems(from: capturedRequest)
        XCTAssertTrue(items.contains(URLQueryItem(name: "category_id", value: "16")))
    }

    func testFetchCampaignsURLIncludesCursor() async throws {
        var capturedRequest: URLRequest?
        MockURLProtocol.requestHandler = { request in
            capturedRequest = request
            return self.makeOKResponse(request: request, body: """
            {"campaigns":[],"next_cursor":null,"total":0}
            """)
        }

        _ = try await client.fetchCampaigns(sort: "trending", categoryID: nil, cursor: "cursor-abc")

        let items = queryItems(from: capturedRequest)
        XCTAssertTrue(items.contains(URLQueryItem(name: "cursor", value: "cursor-abc")))
    }

    func testFetchCampaignsOmitsCategoryIDWhenNil() async throws {
        var capturedRequest: URLRequest?
        MockURLProtocol.requestHandler = { request in
            capturedRequest = request
            return self.makeOKResponse(request: request, body: """
            {"campaigns":[],"next_cursor":null,"total":0}
            """)
        }

        _ = try await client.fetchCampaigns(sort: "trending", categoryID: nil, cursor: nil)

        let items = queryItems(from: capturedRequest)
        XCTAssertFalse(items.contains(where: { $0.name == "category_id" }))
    }

    // MARK: - fetchCampaigns errors

    func testFetchCampaignsThrowsOn500() async {
        MockURLProtocol.requestHandler = { request in
            (HTTPURLResponse(url: request.url!, statusCode: 500, httpVersion: nil, headerFields: nil)!, Data())
        }

        do {
            _ = try await client.fetchCampaigns(sort: "trending", categoryID: nil, cursor: nil)
            XCTFail("Expected throw")
        } catch APIError.serverError(let code) {
            XCTAssertEqual(code, 500)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testFetchCampaignsThrowsOn404() async {
        MockURLProtocol.requestHandler = { request in
            (HTTPURLResponse(url: request.url!, statusCode: 404, httpVersion: nil, headerFields: nil)!, Data())
        }

        do {
            _ = try await client.fetchCampaigns(sort: "trending", categoryID: nil, cursor: nil)
            XCTFail("Expected throw")
        } catch APIError.serverError(let code) {
            XCTAssertEqual(code, 404)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    // MARK: - searchCampaigns

    func testSearchCampaignsURLIncludesQuery() async throws {
        var capturedRequest: URLRequest?
        MockURLProtocol.requestHandler = { request in
            capturedRequest = request
            return self.makeOKResponse(request: request, body: """
            {"campaigns":[],"next_cursor":null}
            """)
        }

        _ = try await client.searchCampaigns(query: "board games", categoryID: nil, cursor: nil)

        let items = queryItems(from: capturedRequest)
        XCTAssertTrue(items.contains(URLQueryItem(name: "q", value: "board games")))
        XCTAssertEqual(capturedRequest?.url?.path, "/api/campaigns/search")
    }

    // MARK: - fetchCategories

    func testFetchCategoriesDecodesResponse() async throws {
        MockURLProtocol.requestHandler = { request in
            self.makeOKResponse(request: request, body: """
            [{"id":"1","name":"Art","parent_id":null}]
            """)
        }

        let categories = try await client.fetchCategories()
        XCTAssertEqual(categories.count, 1)
        XCTAssertEqual(categories.first?.name, "Art")
    }

    // MARK: - deleteAlert

    func testDeleteAlertSendsDELETE() async throws {
        var capturedRequest: URLRequest?
        MockURLProtocol.requestHandler = { request in
            capturedRequest = request
            return (HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!, Data())
        }

        try await client.deleteAlert(id: "alert-99")

        XCTAssertEqual(capturedRequest?.httpMethod, "DELETE")
        XCTAssertTrue(capturedRequest?.url?.path.hasSuffix("alert-99") ?? false)
    }

    func testDeleteAlertThrowsOnError() async {
        MockURLProtocol.requestHandler = { request in
            (HTTPURLResponse(url: request.url!, statusCode: 403, httpVersion: nil, headerFields: nil)!, Data())
        }

        do {
            try await client.deleteAlert(id: "alert-1")
            XCTFail("Expected throw")
        } catch {
            XCTAssertNotNil(error)
        }
    }

    // MARK: - createAlert

    func testCreateAlertSendsPOST() async throws {
        var capturedRequest: URLRequest?
        let alertJSON = """
        {"id":"a1","device_id":"d1","alert_type":"keyword","keyword":"test",
         "category_id":null,"min_percent":0,"velocity_thresh":null,
         "is_enabled":true,"created_at":"2024-01-01T00:00:00Z","last_matched_at":null}
        """
        MockURLProtocol.requestHandler = { request in
            capturedRequest = request
            return self.makeOKResponse(request: request, body: alertJSON)
        }

        let req = CreateAlertRequest(device_id: "d1", alert_type: "keyword", keyword: "test", category_id: nil, min_percent: nil, velocity_thresh: nil)
        _ = try await client.createAlert(req)

        XCTAssertEqual(capturedRequest?.httpMethod, "POST")
        XCTAssertEqual(capturedRequest?.value(forHTTPHeaderField: "Content-Type"), "application/json")
    }

    // MARK: - Helpers

    private func makeOKResponse(request: URLRequest, body: String) -> (HTTPURLResponse, Data) {
        let response = HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!
        return (response, body.data(using: .utf8)!)
    }

    private func queryItems(from request: URLRequest?) -> [URLQueryItem] {
        guard let url = request?.url,
              let components = URLComponents(url: url, resolvingAgainstBaseURL: false) else { return [] }
        return components.queryItems ?? []
    }
}
