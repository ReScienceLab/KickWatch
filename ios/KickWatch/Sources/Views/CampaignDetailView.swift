import SwiftUI
import SwiftData

struct CampaignDetailView: View {
    let campaign: CampaignDTO
    @Query private var watchlist: [Campaign]
    @Environment(\.modelContext) private var modelContext

    private var isWatched: Bool {
        watchlist.contains { $0.pid == campaign.pid && $0.isWatched }
    }

    private var deadline: Date? {
        campaign.deadline.flatMap { ISO8601DateFormatter().date(from: $0) }
    }

    private var daysLeft: Int {
        guard let d = deadline else { return 0 }
        return max(0, Calendar.current.dateComponents([.day], from: .now, to: d).day ?? 0)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                heroImage
                content
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { toolbarItems }
    }

    private var heroImage: some View {
        RemoteImage(urlString: campaign.photo_url ?? "")
            .frame(maxWidth: .infinity)
            .frame(height: 240)
            .clipped()
    }

    private var content: some View {
        VStack(alignment: .leading, spacing: 16) {
            VStack(alignment: .leading, spacing: 4) {
                Text(campaign.name)
                    .font(.title2).fontWeight(.bold)
                if let creator = campaign.creator_name {
                    Text("by \(creator)").font(.subheadline).foregroundStyle(.secondary)
                }
                if let cat = campaign.category_name {
                    Text(cat).font(.caption).padding(.horizontal, 8).padding(.vertical, 3)
                        .background(Color(.systemGray5)).clipShape(Capsule())
                }
            }

            fundingStats

            if let url = campaign.project_url, let link = URL(string: url) {
                Link(destination: link) {
                    Label("Back this project", systemImage: "arrow.up.right.square.fill")
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.accentColor)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                }
            }
        }
        .padding()
    }

    private var fundingStats: some View {
        VStack(spacing: 12) {
            fundingRing
            HStack {
                statBox(label: "Goal", value: formattedAmount(campaign.goal_amount, currency: campaign.goal_currency))
                Divider()
                statBox(label: "Pledged", value: formattedAmount(campaign.pledged_amount, currency: campaign.goal_currency))
                Divider()
                statBox(label: "Days Left", value: "\(daysLeft)")
            }
            .frame(height: 60)
            .padding()
            .background(Color(.systemGray6))
            .clipShape(RoundedRectangle(cornerRadius: 12))
        }
    }

    private var fundingRing: some View {
        let pct = min((campaign.percent_funded ?? 0) / 100, 1.0)
        return ZStack {
            Circle().stroke(Color(.systemGray5), lineWidth: 12)
            Circle().trim(from: 0, to: pct).stroke(Color.accentColor, style: StrokeStyle(lineWidth: 12, lineCap: .round))
                .rotationEffect(.degrees(-90))
            VStack(spacing: 0) {
                Text("\(Int((campaign.percent_funded ?? 0)))%").font(.title2).fontWeight(.bold)
                Text("funded").font(.caption).foregroundStyle(.secondary)
            }
        }
        .frame(width: 120, height: 120)
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
    }

    private func statBox(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(value).font(.subheadline).fontWeight(.semibold)
            Text(label).font(.caption2).foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
    }

    private func formattedAmount(_ amount: Double?, currency: String?) -> String {
        guard let amount else { return "—" }
        let sym = currency == "USD" ? "$" : (currency ?? "")
        if amount >= 1_000_000 { return "\(sym)\(String(format: "%.1fM", amount / 1_000_000))" }
        if amount >= 1_000 { return "\(sym)\(String(format: "%.0fK", amount / 1_000))" }
        return "\(sym)\(Int(amount))"
    }

    @ToolbarContentBuilder
    private var toolbarItems: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            HStack {
                if let url = campaign.project_url, let link = URL(string: url) {
                    ShareLink(item: link)
                }
                Button { toggleWatch() } label: {
                    Image(systemName: isWatched ? "heart.fill" : "heart")
                        .foregroundStyle(isWatched ? .red : .primary)
                }
            }
        }
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
                deadline: deadline ?? .distantFuture,
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
