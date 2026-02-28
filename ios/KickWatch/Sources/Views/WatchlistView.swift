import SwiftUI
import SwiftData

struct WatchlistView: View {
    @Query(filter: #Predicate<Campaign> { $0.isWatched }, sort: \Campaign.deadline)
    private var campaigns: [Campaign]

    @Environment(\.modelContext) private var modelContext

    var body: some View {
        NavigationStack {
            Group {
                if campaigns.isEmpty {
                    emptyState
                } else {
                    List {
                        ForEach(campaigns) { campaign in
                            NavigationLink(destination: CampaignDetailView(campaign: toCampaignDTO(campaign))) {
                                WatchlistRowView(campaign: campaign)
                            }
                        }
                        .onDelete(perform: remove)
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Watchlist")
        }
    }

    private var emptyState: some View {
        VStack(spacing: 16) {
            Image(systemName: "heart.slash").font(.system(size: 48)).foregroundStyle(.secondary)
            Text("No saved campaigns").font(.headline)
            Text("Tap the heart icon on any campaign to add it here.")
                .font(.subheadline).foregroundStyle(.secondary).multilineTextAlignment(.center)
        }
        .padding()
    }

    private func remove(at offsets: IndexSet) {
        for idx in offsets {
            campaigns[idx].isWatched = false
        }
        try? modelContext.save()
    }

    private func toCampaignDTO(_ c: Campaign) -> CampaignDTO {
        CampaignDTO(
            pid: c.pid, name: c.name, blurb: c.blurb, photo_url: c.photoURL,
            goal_amount: c.goalAmount, goal_currency: c.goalCurrency,
            pledged_amount: c.pledgedAmount,
            deadline: ISO8601DateFormatter().string(from: c.deadline),
            state: c.state, category_name: c.categoryName, category_id: c.categoryID,
            project_url: c.projectURL, creator_name: c.creatorName,
            percent_funded: c.percentFunded, backers_count: nil, slug: nil,
            velocity_24h: nil, pledge_delta_24h: nil, first_seen_at: nil
        )
    }
}

struct WatchlistRowView: View {
    let campaign: Campaign

    var body: some View {
        HStack(spacing: 12) {
            RemoteImage(urlString: campaign.photoURL)
                .frame(width: 60, height: 60)
                .clipShape(RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 4) {
                Text(campaign.name).font(.subheadline).fontWeight(.semibold).lineLimit(2)
                Text(campaign.creatorName).font(.caption).foregroundStyle(.secondary)
                HStack {
                    stateBadge
                    Text("\(Int(campaign.percentFunded))% • \(campaign.daysLeft)d left")
                        .font(.caption2).foregroundStyle(.secondary)
                }
            }
        }
        .padding(.vertical, 4)
    }

    private var stateBadge: some View {
        Text(campaign.stateLabel)
            .font(.caption2).fontWeight(.medium)
            .padding(.horizontal, 6).padding(.vertical, 2)
            .background(badgeColor.opacity(0.15))
            .foregroundStyle(badgeColor)
            .clipShape(Capsule())
    }

    private var badgeColor: Color {
        switch campaign.state {
        case "successful": return .green
        case "failed", "canceled": return .red
        default: return .accentColor
        }
    }
}
