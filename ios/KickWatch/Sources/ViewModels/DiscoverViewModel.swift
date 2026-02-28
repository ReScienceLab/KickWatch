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
            print("✅ Loaded \(campaigns.count) campaigns, next_cursor: \(nextCursor ?? "nil"), hasMore: \(hasMore)")
        } catch {
            self.error = error.localizedDescription
            print("❌ Load error: \(error)")
        }
        isLoading = false
    }

    func loadMore() async {
        print("📱 loadMore called - hasMore: \(hasMore), cursor: \(nextCursor ?? "nil"), isLoadingMore: \(isLoadingMore)")
        guard hasMore, let cursor = nextCursor, !isLoadingMore else { 
            print("⏹️ loadMore blocked - hasMore: \(hasMore), cursor: \(nextCursor ?? "nil"), isLoadingMore: \(isLoadingMore)")
            return 
        }
        isLoadingMore = true
        do {
            let resp = try await client.fetchCampaigns(
                sort: selectedSort, categoryID: selectedCategoryID, cursor: cursor
            )
            campaigns.append(contentsOf: resp.campaigns)
            nextCursor = resp.next_cursor
            hasMore = resp.next_cursor != nil
            print("✅ Loaded \(resp.campaigns.count) more campaigns, total: \(campaigns.count), next_cursor: \(nextCursor ?? "nil"), hasMore: \(hasMore)")
        } catch {
            self.error = error.localizedDescription
            print("❌ LoadMore error: \(error)")
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
