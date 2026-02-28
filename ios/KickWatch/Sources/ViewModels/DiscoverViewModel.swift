import Foundation
import SwiftData

@MainActor
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
        isLoadingMore = false  // Cancel any ongoing loadMore
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
        print("📱 loadMore called - hasMore: \(hasMore), cursor: \(nextCursor ?? "nil"), isLoadingMore: \(isLoadingMore), isLoading: \(isLoading)")
        
        // Block loadMore if filter change is in progress OR already loading more
        guard !isLoading, hasMore, let cursor = nextCursor, !isLoadingMore else { 
            print("⏹️ loadMore blocked - isLoading: \(isLoading), hasMore: \(hasMore), cursor: \(nextCursor ?? "nil"), isLoadingMore: \(isLoadingMore)")
            return 
        }
        
        isLoadingMore = true
        do {
            let resp = try await client.fetchCampaigns(
                sort: selectedSort, categoryID: selectedCategoryID, cursor: cursor
            )
            
            // Deduplicate by PID before appending
            let existingPIDs = Set(campaigns.map(\.pid))
            let newCampaigns = resp.campaigns.filter { !existingPIDs.contains($0.pid) }
            
            campaigns.append(contentsOf: newCampaigns)
            nextCursor = resp.next_cursor
            hasMore = resp.next_cursor != nil
            print("✅ Loaded \(resp.campaigns.count) more campaigns (\(newCampaigns.count) new), total: \(campaigns.count), next_cursor: \(nextCursor ?? "nil"), hasMore: \(hasMore)")
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
        guard selectedSort != sort else { return }  // Prevent redundant reloads
        selectedSort = sort
        await load()
    }

    func selectCategory(_ id: String?) async {
        guard selectedCategoryID != id else { return }  // Prevent redundant reloads
        selectedCategoryID = id
        await load()
    }
}
