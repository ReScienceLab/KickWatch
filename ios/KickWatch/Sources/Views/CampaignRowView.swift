import SwiftUI

struct CampaignRowView: View {
    let campaign: CampaignDTO
    @Query private var watchlist: [Campaign]

    private var isWatched: Bool {
        watchlist.contains { $0.pid == campaign.pid && $0.isWatched }
    }

    var body: some View {
        HStack(spacing: 12) {
            thumbnail
            info
            Spacer()
            watchButton
        }
        .padding(.vertical, 10)
        .padding(.leading, 16)
    }

    private var thumbnail: some View {
        RemoteImage(urlString: campaign.photo_url ?? "")
            .frame(width: 72, height: 72)
            .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    private var info: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(campaign.name)
                .font(.subheadline).fontWeight(.semibold)
                .lineLimit(2)
            if let creator = campaign.creator_name {
                Text("by \(creator)")
                    .font(.caption).foregroundStyle(.secondary)
            }
            fundingBar
            HStack(spacing: 8) {
                Text("\(Int(campaign.percent_funded ?? 0))% funded")
                    .font(.caption2).foregroundStyle(.secondary)
                if let deadline = campaign.deadline, let date = ISO8601DateFormatter().date(from: deadline) {
                    let days = max(0, Calendar.current.dateComponents([.day], from: .now, to: date).day ?? 0)
                    Text("\(days)d left")
                        .font(.caption2).foregroundStyle(.secondary)
                }
            }
        }
    }

    private var fundingBar: some View {
        GeometryReader { geo in
            ZStack(alignment: .leading) {
                RoundedRectangle(cornerRadius: 2).fill(Color(.systemGray5)).frame(height: 4)
                RoundedRectangle(cornerRadius: 2).fill(Color.accentColor)
                    .frame(width: min(geo.size.width * CGFloat((campaign.percent_funded ?? 0) / 100), geo.size.width), height: 4)
            }
        }
        .frame(height: 4)
    }

    @Environment(\.modelContext) private var modelContext

    private var watchButton: some View {
        Button {
            toggleWatch()
        } label: {
            Image(systemName: isWatched ? "heart.fill" : "heart")
                .foregroundStyle(isWatched ? .red : .secondary)
        }
        .buttonStyle(.plain)
    }

    private func toggleWatch() {
        if let existing = watchlist.first(where: { $0.pid == campaign.pid }) {
            existing.isWatched.toggle()
        } else {
            let c = Campaign(
                pid: campaign.pid,
                name: campaign.name,
                blurb: campaign.blurb ?? "",
                photoURL: campaign.photo_url ?? "",
                goalAmount: campaign.goal_amount ?? 0,
                goalCurrency: campaign.goal_currency ?? "USD",
                pledgedAmount: campaign.pledged_amount ?? 0,
                state: campaign.state ?? "live",
                categoryName: campaign.category_name ?? "",
                categoryID: campaign.category_id ?? "",
                projectURL: campaign.project_url ?? "",
                creatorName: campaign.creator_name ?? "",
                percentFunded: campaign.percent_funded ?? 0,
                isWatched: true
            )
            modelContext.insert(c)
        }
        try? modelContext.save()
    }
}
