import SwiftUI
import SwiftData

struct SearchView: View {
    @State private var query = ""
    @State private var results: [CampaignDTO] = []
    @State private var isLoading = false
    @State private var isLoadingMore = false
    @State private var nextCursor: String?
    @State private var hasMore = false
    @Environment(\.modelContext) private var modelContext
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                if isLoading && results.isEmpty {
                    ProgressView().frame(maxWidth: .infinity)
                } else {
                    ForEach(results, id: \.pid) { campaign in
                        NavigationLink(destination: CampaignDetailView(campaign: campaign)) {
                            CampaignRowView(campaign: campaign)
                        }
                        .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 16))
                    }
                    if isLoadingMore {
                        ProgressView().frame(maxWidth: .infinity)
                    } else if hasMore {
                        Color.clear
                            .frame(height: 1)
                            .task { await loadMore() }
                    }
                }
            }
            .listStyle(.plain)
            .navigationTitle("Search")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } } }
            .searchable(text: $query, placement: .navigationBarDrawer(displayMode: .always))
            .onSubmit(of: .search) { Task { await search() } }
            .onChange(of: query) { _, new in if new.isEmpty { results = [] } }
        }
    }

    private func search() async {
        guard !query.isEmpty else { return }
        isLoading = true
        isLoadingMore = false  // Cancel any ongoing loadMore
        do {
            let resp = try await APIClient.shared.searchCampaigns(query: query)
            results = resp.campaigns
            nextCursor = resp.next_cursor
            hasMore = resp.next_cursor != nil
        } catch {
            print("SearchView: \(error)")
        }
        isLoading = false
    }

    private func loadMore() async {
        guard !isLoading, !isLoadingMore, let cursor = nextCursor else { return }
        isLoadingMore = true
        do {
            let resp = try await APIClient.shared.searchCampaigns(query: query, cursor: cursor)
            
            // Deduplicate by PID before appending
            let existingPIDs = Set(results.map(\.pid))
            let newResults = resp.campaigns.filter { !existingPIDs.contains($0.pid) }
            
            results.append(contentsOf: newResults)
            nextCursor = resp.next_cursor
            hasMore = resp.next_cursor != nil
        } catch {
            print("SearchView loadMore: \(error)")
        }
        isLoadingMore = false
    }
}
