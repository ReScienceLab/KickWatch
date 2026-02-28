import XCTest
@testable import KickWatch

final class DTOTests: XCTestCase {
    private let decoder = JSONDecoder()
    private let encoder = JSONEncoder()

    // MARK: - CampaignDTO

    func testCampaignDTODecodeFullFields() throws {
        let json = """
        {
            "pid": "123", "name": "Cool Project", "blurb": "A blurb",
            "photo_url": "https://example.com/img.jpg",
            "goal_amount": 5000.0, "goal_currency": "USD",
            "pledged_amount": 2500.0, "deadline": "2024-12-31T00:00:00Z",
            "state": "live", "category_name": "Technology", "category_id": "16",
            "project_url": "https://kickstarter.com/projects/test",
            "creator_name": "Alice", "percent_funded": 50.0,
            "slug": "test-project", "velocity_24h": 100.0,
            "pledge_delta_24h": 50.0, "first_seen_at": "2024-01-01T00:00:00Z"
        }
        """.data(using: .utf8)!

        let dto = try decoder.decode(CampaignDTO.self, from: json)
        XCTAssertEqual(dto.pid, "123")
        XCTAssertEqual(dto.name, "Cool Project")
        XCTAssertEqual(dto.blurb, "A blurb")
        XCTAssertEqual(dto.goal_amount, 5000.0)
        XCTAssertEqual(dto.goal_currency, "USD")
        XCTAssertEqual(dto.pledged_amount, 2500.0)
        XCTAssertEqual(dto.state, "live")
        XCTAssertEqual(dto.category_name, "Technology")
        XCTAssertEqual(dto.percent_funded, 50.0)
        XCTAssertEqual(dto.slug, "test-project")
        XCTAssertEqual(dto.velocity_24h, 100.0)
        XCTAssertEqual(dto.pledge_delta_24h, 50.0)
    }

    func testCampaignDTODecodeMinimalFields() throws {
        let json = #"{"pid": "42", "name": "Minimal"}"#.data(using: .utf8)!

        let dto = try decoder.decode(CampaignDTO.self, from: json)
        XCTAssertEqual(dto.pid, "42")
        XCTAssertEqual(dto.name, "Minimal")
        XCTAssertNil(dto.blurb)
        XCTAssertNil(dto.photo_url)
        XCTAssertNil(dto.deadline)
        XCTAssertNil(dto.velocity_24h)
    }

    // MARK: - CategoryDTO

    func testCategoryDTODecodeNoParent() throws {
        let json = #"{"id": "16", "name": "Technology", "parent_id": null}"#.data(using: .utf8)!

        let dto = try decoder.decode(CategoryDTO.self, from: json)
        XCTAssertEqual(dto.id, "16")
        XCTAssertEqual(dto.name, "Technology")
        XCTAssertNil(dto.parent_id)
    }

    func testCategoryDTODecodeWithParent() throws {
        let json = #"{"id": "44", "name": "Apps", "parent_id": "16"}"#.data(using: .utf8)!

        let dto = try decoder.decode(CategoryDTO.self, from: json)
        XCTAssertEqual(dto.parent_id, "16")
    }

    // MARK: - CampaignListResponse

    func testCampaignListResponseWithCursor() throws {
        let json = """
        {"campaigns": [{"pid": "1", "name": "P1"}], "next_cursor": "abc123", "total": 42}
        """.data(using: .utf8)!

        let resp = try decoder.decode(CampaignListResponse.self, from: json)
        XCTAssertEqual(resp.campaigns.count, 1)
        XCTAssertEqual(resp.next_cursor, "abc123")
        XCTAssertEqual(resp.total, 42)
    }

    func testCampaignListResponseNoCursor() throws {
        let json = #"{"campaigns": [], "next_cursor": null, "total": 0}"#.data(using: .utf8)!

        let resp = try decoder.decode(CampaignListResponse.self, from: json)
        XCTAssertNil(resp.next_cursor)
        XCTAssertEqual(resp.total, 0)
    }

    // MARK: - SearchResponse

    func testSearchResponseDecode() throws {
        let json = #"{"campaigns": [], "next_cursor": null}"#.data(using: .utf8)!

        let resp = try decoder.decode(SearchResponse.self, from: json)
        XCTAssertTrue(resp.campaigns.isEmpty)
        XCTAssertNil(resp.next_cursor)
    }

    func testSearchResponseWithResults() throws {
        let json = """
        {"campaigns": [{"pid": "x", "name": "X"}], "next_cursor": "next"}
        """.data(using: .utf8)!

        let resp = try decoder.decode(SearchResponse.self, from: json)
        XCTAssertEqual(resp.campaigns.count, 1)
        XCTAssertEqual(resp.next_cursor, "next")
    }

    // MARK: - AlertDTO

    func testAlertDTODecode() throws {
        let json = """
        {
            "id": "alert-1", "device_id": "device-abc", "alert_type": "keyword",
            "keyword": "gaming", "category_id": null, "min_percent": 0.0,
            "velocity_thresh": null, "is_enabled": true,
            "created_at": "2024-01-01T00:00:00Z", "last_matched_at": null
        }
        """.data(using: .utf8)!

        let dto = try decoder.decode(AlertDTO.self, from: json)
        XCTAssertEqual(dto.id, "alert-1")
        XCTAssertEqual(dto.device_id, "device-abc")
        XCTAssertEqual(dto.keyword, "gaming")
        XCTAssertTrue(dto.is_enabled)
        XCTAssertNil(dto.last_matched_at)
        XCTAssertNil(dto.velocity_thresh)
    }

    func testAlertDTODecodeWithOptionalsFilled() throws {
        let json = """
        {
            "id": "a2", "device_id": "d1", "alert_type": "category",
            "keyword": "", "category_id": "16", "min_percent": 50.0,
            "velocity_thresh": 100.0, "is_enabled": false,
            "created_at": "2024-06-01T00:00:00Z", "last_matched_at": "2024-06-15T00:00:00Z"
        }
        """.data(using: .utf8)!

        let dto = try decoder.decode(AlertDTO.self, from: json)
        XCTAssertEqual(dto.category_id, "16")
        XCTAssertEqual(dto.min_percent, 50.0)
        XCTAssertEqual(dto.velocity_thresh, 100.0)
        XCTAssertFalse(dto.is_enabled)
        XCTAssertNotNil(dto.last_matched_at)
    }

    // MARK: - Encoding requests

    func testCreateAlertRequestEncoding() throws {
        let req = CreateAlertRequest(
            device_id: "d1", alert_type: "keyword",
            keyword: "board games", category_id: nil,
            min_percent: 25.0, velocity_thresh: nil
        )
        let data = try encoder.encode(req)
        let obj = try JSONSerialization.jsonObject(with: data) as! [String: Any]
        XCTAssertEqual(obj["device_id"] as? String, "d1")
        XCTAssertEqual(obj["keyword"] as? String, "board games")
        XCTAssertEqual(obj["min_percent"] as? Double, 25.0)
    }

    func testUpdateAlertRequestEncoding() throws {
        let req = UpdateAlertRequest(is_enabled: false, keyword: nil, category_id: nil, min_percent: nil)
        let data = try encoder.encode(req)
        let obj = try JSONSerialization.jsonObject(with: data) as! [String: Any]
        XCTAssertEqual(obj["is_enabled"] as? Bool, false)
    }

    func testRegisterDeviceRequestEncoding() throws {
        let req = RegisterDeviceRequest(device_token: "tok123")
        let data = try encoder.encode(req)
        let obj = try JSONSerialization.jsonObject(with: data) as! [String: Any]
        XCTAssertEqual(obj["device_token"] as? String, "tok123")
    }

    // MARK: - APIError

    func testAPIErrorInvalidURLDescription() {
        XCTAssertEqual(APIError.invalidURL.errorDescription, "Invalid URL")
    }

    func testAPIErrorInvalidResponseDescription() {
        XCTAssertEqual(APIError.invalidResponse.errorDescription, "Invalid server response")
    }

    func testAPIErrorServerError404Description() {
        XCTAssertEqual(APIError.serverError(statusCode: 404).errorDescription, "Server error: 404")
    }

    func testAPIErrorServerError500Description() {
        XCTAssertEqual(APIError.serverError(statusCode: 500).errorDescription, "Server error: 500")
    }
}
