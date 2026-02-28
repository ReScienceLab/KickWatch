import Foundation
@testable import KickWatch

final class MockAPIClient: APIClientProtocol, @unchecked Sendable {
    // Stubbed responses
    var campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: nil)
    var searchResponse = SearchResponse(campaigns: [], next_cursor: nil)
    var categoriesResponse: [CategoryDTO] = []
    var alertsResponse: [AlertDTO] = []
    var createAlertResult: AlertDTO = MockAPIClient.makeAlertDTO()
    var updateAlertResult: AlertDTO = MockAPIClient.makeAlertDTO()
    var alertMatchesResult: [CampaignDTO] = []
    var registerDeviceResult = RegisterDeviceResponse(device_id: "mock-device-id")

    // Error stub
    var shouldThrow: Error?

    // Call tracking
    var fetchCampaignsCalls: [(sort: String, categoryID: String?, cursor: String?)] = []
    var searchCalls: [(query: String, categoryID: String?, cursor: String?)] = []
    var fetchCategoriesCalled = false
    var deleteAlertIDs: [String] = []
    var createAlertRequests: [CreateAlertRequest] = []
    var updateAlertRequests: [(id: String, req: UpdateAlertRequest)] = []
    var fetchAlertsCalled = false

    func fetchCampaigns(sort: String, categoryID: String?, cursor: String?) async throws -> CampaignListResponse {
        fetchCampaignsCalls.append((sort, categoryID, cursor))
        if let e = shouldThrow { throw e }
        return campaignListResponse
    }

    func searchCampaigns(query: String, categoryID: String?, cursor: String?) async throws -> SearchResponse {
        searchCalls.append((query, categoryID, cursor))
        if let e = shouldThrow { throw e }
        return searchResponse
    }

    func fetchCategories() async throws -> [CategoryDTO] {
        fetchCategoriesCalled = true
        if let e = shouldThrow { throw e }
        return categoriesResponse
    }

    func registerDevice(token: String) async throws -> RegisterDeviceResponse {
        if let e = shouldThrow { throw e }
        return registerDeviceResult
    }

    func fetchAlerts(deviceID: String) async throws -> [AlertDTO] {
        fetchAlertsCalled = true
        if let e = shouldThrow { throw e }
        return alertsResponse
    }

    func createAlert(_ req: CreateAlertRequest) async throws -> AlertDTO {
        createAlertRequests.append(req)
        if let e = shouldThrow { throw e }
        return createAlertResult
    }

    func updateAlert(id: String, req: UpdateAlertRequest) async throws -> AlertDTO {
        updateAlertRequests.append((id, req))
        if let e = shouldThrow { throw e }
        return updateAlertResult
    }

    func deleteAlert(id: String) async throws {
        deleteAlertIDs.append(id)
        if let e = shouldThrow { throw e }
    }

    func fetchAlertMatches(alertID: String) async throws -> [CampaignDTO] {
        if let e = shouldThrow { throw e }
        return alertMatchesResult
    }

    // MARK: - Factories

    static func makeAlertDTO(id: String = "alert-1", isEnabled: Bool = true) -> AlertDTO {
        AlertDTO(
            id: id,
            device_id: "device-1",
            alert_type: "keyword",
            keyword: "test",
            category_id: nil,
            min_percent: 0,
            velocity_thresh: nil,
            is_enabled: isEnabled,
            created_at: "2024-01-01T00:00:00Z",
            last_matched_at: nil
        )
    }

    static func makeCampaignDTO(pid: String = "c1", name: String = "Test Campaign") -> CampaignDTO {
        CampaignDTO(
            pid: pid,
            name: name,
            blurb: nil,
            photo_url: nil,
            goal_amount: 1000,
            goal_currency: "USD",
            pledged_amount: 500,
            deadline: nil,
            state: "live",
            category_name: nil,
            category_id: nil,
            project_url: nil,
            creator_name: nil,
            percent_funded: 50,
            slug: nil,
            velocity_24h: nil,
            pledge_delta_24h: nil,
            first_seen_at: nil
        )
    }
}
