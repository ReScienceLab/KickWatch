import Foundation

struct CampaignDTO: Codable {
    let pid: String
    let name: String
    let blurb: String?
    let photo_url: String?
    let goal_amount: Double?
    let goal_currency: String?
    let pledged_amount: Double?
    let deadline: String?
    let state: String?
    let category_name: String?
    let category_id: String?
    let project_url: String?
    let creator_name: String?
    let percent_funded: Double?
    let slug: String?
    let velocity_24h: Double?
    let pledge_delta_24h: Double?
    let first_seen_at: String?
}

struct CategoryDTO: Codable {
    let id: String
    let name: String
    let parent_id: String?
}

struct CampaignListResponse: Codable {
    let campaigns: [CampaignDTO]
    let next_cursor: String?
    let total: Int?
}

struct SearchResponse: Codable {
    let campaigns: [CampaignDTO]
    let next_cursor: String?
}

struct CampaignSnapshotDTO: Codable {
    let campaign_pid: String
    let snapshot_date: String
    let pledged_amount: Double
    let percent_funded: Double
    let backers_count: Int?
}

struct RegisterDeviceRequest: Codable {
    let device_token: String
}

struct RegisterDeviceResponse: Codable {
    let device_id: String
}

struct CreateAlertRequest: Codable {
    let device_id: String
    let alert_type: String?
    let keyword: String?
    let category_id: String?
    let min_percent: Double?
    let velocity_thresh: Double?
}

struct AlertDTO: Codable {
    let id: String
    let device_id: String
    let alert_type: String?
    let keyword: String
    let category_id: String?
    let min_percent: Double
    let velocity_thresh: Double?
    let is_enabled: Bool
    let created_at: String
    let last_matched_at: String?
}

struct UpdateAlertRequest: Codable {
    let is_enabled: Bool?
    let keyword: String?
    let category_id: String?
    let min_percent: Double?
}

enum APIError: LocalizedError {
    case invalidURL
    case invalidResponse
    case serverError(statusCode: Int)

    var errorDescription: String? {
        switch self {
        case .invalidURL: return "Invalid URL"
        case .invalidResponse: return "Invalid server response"
        case .serverError(let code): return "Server error: \(code)"
        }
    }
}

actor APIClient {
    static let shared = APIClient()

    private let baseURL: String
    private let session: URLSession

    init(baseURL: String? = nil) {
        #if DEBUG
        self.baseURL = baseURL ?? "https://api-dev.kickwatch.rescience.com"
        #else
        self.baseURL = baseURL ?? "https://api.kickwatch.rescience.com"
        #endif
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        self.session = URLSession(configuration: config)
    }

    func fetchCampaigns(sort: String = "trending", categoryID: String? = nil, cursor: String? = nil) async throws -> CampaignListResponse {
        var components = URLComponents(string: baseURL + "/api/campaigns")!
        var items: [URLQueryItem] = [URLQueryItem(name: "sort", value: sort)]
        if let cat = categoryID { items.append(URLQueryItem(name: "category_id", value: cat)) }
        if let cur = cursor { items.append(URLQueryItem(name: "cursor", value: cur)) }
        components.queryItems = items
        return try await get(url: components.url!)
    }

    func searchCampaigns(query: String, categoryID: String? = nil, cursor: String? = nil) async throws -> SearchResponse {
        var components = URLComponents(string: baseURL + "/api/campaigns/search")!
        var items: [URLQueryItem] = [URLQueryItem(name: "q", value: query)]
        if let cat = categoryID { items.append(URLQueryItem(name: "category_id", value: cat)) }
        if let cur = cursor { items.append(URLQueryItem(name: "cursor", value: cur)) }
        components.queryItems = items
        return try await get(url: components.url!)
    }

    func fetchCategories() async throws -> [CategoryDTO] {
        return try await get(url: URL(string: baseURL + "/api/categories")!)
    }

    func registerDevice(token: String) async throws -> RegisterDeviceResponse {
        return try await post(url: URL(string: baseURL + "/api/devices/register")!, body: RegisterDeviceRequest(device_token: token))
    }

    func fetchAlerts(deviceID: String) async throws -> [AlertDTO] {
        let url = URL(string: baseURL + "/api/alerts?device_id=\(deviceID)")!
        return try await get(url: url)
    }

    func createAlert(_ req: CreateAlertRequest) async throws -> AlertDTO {
        return try await post(url: URL(string: baseURL + "/api/alerts")!, body: req)
    }

    func updateAlert(id: String, req: UpdateAlertRequest) async throws -> AlertDTO {
        return try await patch(url: URL(string: baseURL + "/api/alerts/\(id)")!, body: req)
    }

    func deleteAlert(id: String) async throws {
        let url = URL(string: baseURL + "/api/alerts/\(id)")!
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, 200..<300 ~= http.statusCode else {
            throw APIError.invalidResponse
        }
    }

    func fetchCampaignHistory(pid: String) async throws -> [CampaignSnapshotDTO] {
        return try await get(url: URL(string: baseURL + "/api/campaigns/\(pid)/history")!)
    }

    func fetchAlertMatches(alertID: String) async throws -> [CampaignDTO] {
        let url = URL(string: baseURL + "/api/alerts/\(alertID)/matches")!
        return try await get(url: url)
    }

    private func get<R: Decodable>(url: URL) async throws -> R {
        let (data, response) = try await session.data(from: url)
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }
        guard 200..<300 ~= http.statusCode else { throw APIError.serverError(statusCode: http.statusCode) }
        return try JSONDecoder().decode(R.self, from: data)
    }

    private func post<T: Encodable, R: Decodable>(url: URL, body: T) async throws -> R {
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }
        guard 200..<300 ~= http.statusCode else { throw APIError.serverError(statusCode: http.statusCode) }
        return try JSONDecoder().decode(R.self, from: data)
    }

    private func patch<T: Encodable, R: Decodable>(url: URL, body: T) async throws -> R {
        var request = URLRequest(url: url)
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }
        guard 200..<300 ~= http.statusCode else { throw APIError.serverError(statusCode: http.statusCode) }
        return try JSONDecoder().decode(R.self, from: data)
    }
}
