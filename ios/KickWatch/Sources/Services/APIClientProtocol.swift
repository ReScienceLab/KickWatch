import Foundation

protocol APIClientProtocol: Sendable {
    func fetchCampaigns(sort: String, categoryID: String?, cursor: String?) async throws -> CampaignListResponse
    func searchCampaigns(query: String, categoryID: String?, cursor: String?) async throws -> SearchResponse
    func fetchCategories() async throws -> [CategoryDTO]
    func registerDevice(token: String) async throws -> RegisterDeviceResponse
    func fetchAlerts(deviceID: String) async throws -> [AlertDTO]
    func createAlert(_ req: CreateAlertRequest) async throws -> AlertDTO
    func updateAlert(id: String, req: UpdateAlertRequest) async throws -> AlertDTO
    func deleteAlert(id: String) async throws
    func fetchAlertMatches(alertID: String) async throws -> [CampaignDTO]
}
