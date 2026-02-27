import Foundation
import SwiftData

@Observable
final class DiscoverViewModel {
    private let client: any APIClientProtocol

    var campaigns: [CampaignDTO] = []
    var categories: [CategoryDTO] = []
    var isLoading = false
    var isLoadingMore = false
    var error: String?
    var nextCursor: String?
    var hasMore = false

    var selectedSort = "trending"
    var selectedCategoryID: String?

    init(client: any APIClientProtocol = APIClient.shared) {
        self.client = client
    }

    func load() async {
        isLoading = true
        error = nil
        do {
            let resp = try await client.fetchCampaigns(
                sort: selectedSort, categoryID: selectedCategoryID, cursor: nil
            )
            campaigns = resp.campaigns
            nextCursor = resp.next_cursor
            hasMore = resp.next_cursor != nil
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    func loadMore() async {
        guard hasMore, let cursor = nextCursor, !isLoadingMore else { return }
        isLoadingMore = true
        do {
            let resp = try await client.fetchCampaigns(
                sort: selectedSort, categoryID: selectedCategoryID, cursor: cursor
            )
            campaigns.append(contentsOf: resp.campaigns)
            nextCursor = resp.next_cursor
            hasMore = resp.next_cursor != nil
        } catch {
            self.error = error.localizedDescription
        }
        isLoadingMore = false
    }

    func loadCategories() async {
        guard categories.isEmpty else { return }
        do {
            categories = try await client.fetchCategories()
        } catch {
            print("DiscoverViewModel: failed to load categories: \(error)")
        }
    }

    func selectSort(_ sort: String) async {
        selectedSort = sort
        nextCursor = nil
        await load()
    }

    func selectCategory(_ id: String?) async {
        selectedCategoryID = id
        nextCursor = nil
        await load()
    }
}
