import SwiftUI
import SwiftData

struct CampaignDetailView: View {
    let campaign: CampaignDTO
    @Query private var watchlist: [Campaign]
    @Environment(\.modelContext) private var modelContext
    @State private var historyData: [HistoryDataPoint] = []
    @State private var isLoadingHistory = false

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
        .task {
            await loadHistory()
        }
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
            .padding(.horizontal)

            fundingStats
                .padding(.horizontal)

            if let blurb = campaign.blurb, !blurb.isEmpty {
                ExpandableBlurbView(blurb: blurb)
                    .padding(.horizontal)
            }

            // Show momentum section if we have history data OR if there's 24h activity
            if !historyData.isEmpty ||
               (campaign.velocity_24h != nil && campaign.pledge_delta_24h != nil &&
                (campaign.velocity_24h! > 0 || campaign.pledge_delta_24h! != 0)) {
                momentumSection(
                    velocity: campaign.velocity_24h ?? 0,
                    delta: campaign.pledge_delta_24h ?? 0
                )
                .padding(.horizontal)
            }

            if let url = campaign.project_url, let link = URL(string: url) {
                Link(destination: link) {
                    Label("Back this project", systemImage: "arrow.up.right.square.fill")
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.accentColor)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                }
                .padding(.horizontal)
            }
        }
    }

    private var fundingStats: some View {
        VStack(spacing: 12) {
            fundingRing

            VStack(spacing: 8) {
                HStack {
                    statBox(label: "Goal",
                           value: formattedAmount(campaign.goal_amount, currency: campaign.goal_currency),
                           icon: "target")
                    Divider()
                    statBox(label: "Pledged",
                           value: formattedAmount(campaign.pledged_amount, currency: campaign.goal_currency),
                           icon: "dollarsign.circle.fill")
                }
                .frame(height: 60)

                HStack {
                    statBox(label: "Backers",
                           value: formatBackersLarge(campaign.backers_count),
                           icon: "person.2.fill")
                    Divider()
                    statBox(label: "Days Left",
                           value: "\(daysLeft)",
                           icon: "calendar")
                }
                .frame(height: 60)
            }
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
                Text(formatPercentFunded(campaign.percent_funded ?? 0))
                    .font(.title2).fontWeight(.bold)
                Text("funded").font(.caption).foregroundStyle(.secondary)
            }
        }
        .frame(width: 120, height: 120)
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
    }

    private func statBox(label: String, value: String, icon: String? = nil) -> some View {
        VStack(spacing: 4) {
            if let icon = icon {
                Image(systemName: icon)
                    .font(.system(size: 16))
                    .foregroundStyle(.secondary)
            }
            Text(value).font(.title3).fontWeight(.bold)
            Text(label).font(.caption).foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
    }

    private func formatBackersLarge(_ count: Int?) -> String {
        guard let count = count else { return "—" }
        if count >= 1_000_000 {
            return String(format: "%.1fM", Double(count) / 1_000_000)
        } else if count >= 1_000 {
            return String(format: "%.1fK", Double(count) / 1_000)
        }
        return "\(count)"
    }

    private func formattedAmount(_ amount: Double?, currency: String?) -> String {
        guard let amount else { return "—" }
        let sym = currency == "USD" ? "$" : (currency ?? "")
        if amount >= 1_000_000 { return "\(sym)\(String(format: "%.1fM", amount / 1_000_000))" }
        if amount >= 1_000 { return "\(sym)\(String(format: "%.0fK", amount / 1_000))" }
        return "\(sym)\(Int(amount))"
    }

    private func formatPercentFunded(_ percent: Double) -> String {
        // For very high percentages, format as "18.9K%" for readability
        if percent >= 10_000 {
            return String(format: "%.1fK%%", percent / 1_000)
        } else if percent >= 1_000 {
            return String(format: "%.2fK%%", percent / 1_000)
        }
        return String(format: "%.0f%%", percent)
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

    private func momentumSection(velocity: Double, delta: Double) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: "bolt.fill")
                    .foregroundStyle(.orange)
                Text("24-Hour Momentum")
                    .font(.subheadline).fontWeight(.semibold)
                Spacer()
                momentumBadge(velocity: velocity, delta: delta)
            }

            if !historyData.isEmpty {
                SparklineView(dataPoints: historyData)
            } else if isLoadingHistory {
                ProgressView()
                    .frame(height: 60)
                    .frame(maxWidth: .infinity)
            }

            HStack(spacing: 16) {
                metricCard(
                    icon: "dollarsign.circle.fill",
                    label: "24h Change",
                    value: formatDelta(delta),
                    color: delta > 0 ? .green : (delta < 0 ? .red : .gray)
                )
                metricCard(
                    icon: "percent",
                    label: "Growth Rate",
                    value: String(format: "%.1f%%", velocity),
                    color: velocity > 0 ? .green : (velocity < 0 ? .red : .gray)
                )
            }
        }
        .padding()
        .background(Color(.systemGray6))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    private func momentumBadge(velocity: Double, delta: Double) -> some View {
        let icon = velocity > 0 ? "arrow.up.right" : (velocity < 0 ? "arrow.down.right" : "arrow.right")
        let color: Color = velocity > 0 ? .green : (velocity < 0 ? .red : .gray)

        return HStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 10))
            Text(delta > 0 ? "+\(formatDelta(delta))" : formatDelta(delta))
                .font(.caption).fontWeight(.semibold)
        }
        .padding(.horizontal, 8).padding(.vertical, 4)
        .background(color.opacity(0.15))
        .foregroundStyle(color)
        .clipShape(Capsule())
    }

    private func metricCard(icon: String, label: String, value: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 14))
                Text(label)
                    .font(.caption2)
            }
            .foregroundStyle(.secondary)

            Text(value)
                .font(.title3).fontWeight(.bold)
                .foregroundStyle(color)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(Color(.systemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    private func formatDelta(_ amount: Double) -> String {
        let absAmount = abs(amount)
        if absAmount >= 1_000_000 {
            return String(format: "$%.1fM", amount / 1_000_000)
        } else if absAmount >= 1_000 {
            return String(format: "$%.0fK", amount / 1_000)
        }
        return "$\(Int(amount))"
    }

    private func loadHistory() async {
        isLoadingHistory = true
        defer { isLoadingHistory = false }

        do {
            let response = try await APIClient.shared.fetchCampaignHistory(pid: campaign.pid, days: 14)
            historyData = response.history
        } catch {
            print("Failed to load history: \(error)")
        }
    }
}

struct ExpandableBlurbView: View {
    let blurb: String
    @State private var isExpanded = false

    private var shouldShowButton: Bool {
        blurb.count > 150
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("About this project")
                .font(.subheadline).fontWeight(.semibold)

            Text(blurb)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineLimit(isExpanded ? nil : 3)

            if shouldShowButton {
                Button {
                    withAnimation(.easeInOut(duration: 0.2)) {
                        isExpanded.toggle()
                    }
                } label: {
                    HStack(spacing: 4) {
                        Text(isExpanded ? "Show less" : "Read more")
                        Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                            .font(.system(size: 10))
                    }
                    .font(.caption).fontWeight(.medium)
                    .foregroundStyle(Color.accentColor)
                }
            }
        }
        .padding()
        .background(Color(.systemGray6))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}
