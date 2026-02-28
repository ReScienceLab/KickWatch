import SwiftUI
import SwiftData

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
            Spacer(minLength: 4)
            watchButton
                .padding(.trailing, 16)
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
                stateBadge
                Text("\(formatPercentFunded(campaign.percent_funded ?? 0)) funded")
                    .font(.caption2).foregroundStyle(.secondary)
                if let deadline = campaign.deadline, let date = ISO8601DateFormatter().date(from: deadline) {
                    let days = max(0, Calendar.current.dateComponents([.day], from: .now, to: date).day ?? 0)
                    Text("\(days)d left")
                        .font(.caption2).foregroundStyle(.secondary)
                }
                if let backers = campaign.backers_count, backers > 0 {
                    HStack(spacing: 2) {
                        Image(systemName: "person.2.fill")
                            .font(.system(size: 9))
                        Text(formatBackers(backers))
                    }
                    .font(.caption2).foregroundStyle(.secondary)
                }
                momentumBadge
            }
        }
    }

    @ViewBuilder
    private var momentumBadge: some View {
        if let v = campaign.velocity_24h, v > 0 {
            let (icon, color): (String, Color) = v >= 200 ? ("🔥", .red) : ("⚡", .orange)

            if let delta = campaign.pledge_delta_24h, delta > 0 {
                Text("\(icon) +\(formatDelta(delta))")
                    .font(.caption2).fontWeight(.semibold)
                    .foregroundStyle(color)
            } else {
                Text("\(icon) +\(Int(v))%")
                    .font(.caption2).fontWeight(.semibold)
                    .foregroundStyle(color)
            }
        } else if isNew {
            Text("New")
                .font(.caption2).fontWeight(.semibold)
                .padding(.horizontal, 5).padding(.vertical, 2)
                .background(Color.blue.opacity(0.15))
                .foregroundStyle(.blue)
                .clipShape(Capsule())
        }
    }

    @ViewBuilder
    private var stateBadge: some View {
        if let state = campaign.state, state != "live" {
            Text(stateLabel(state))
                .font(.caption2).fontWeight(.medium)
                .padding(.horizontal, 6).padding(.vertical, 2)
                .background(stateColor(state).opacity(0.15))
                .foregroundStyle(stateColor(state))
                .clipShape(Capsule())
        }
    }

    private var isNew: Bool {
        guard let s = campaign.first_seen_at,
              let date = ISO8601DateFormatter().date(from: s) else { return false }
        return Date().timeIntervalSince(date) < 48 * 3600
    }

    private func formatBackers(_ count: Int) -> String {
        if count >= 1000 {
            return String(format: "%.1fK", Double(count) / 1000)
        }
        return "\(count)"
    }

    private func formatDelta(_ amount: Double) -> String {
        if amount >= 1_000_000 {
            return String(format: "$%.1fM", amount / 1_000_000)
        } else if amount >= 1_000 {
            return String(format: "$%.0fK", amount / 1_000)
        }
        return "$\(Int(amount))"
    }

    private func formatPercentFunded(_ percent: Double) -> String {
        if percent >= 10_000 {
            return String(format: "%.1fK%%", percent / 1_000)
        } else if percent >= 1_000 {
            return String(format: "%.2fK%%", percent / 1_000)
        }
        return String(format: "%.0f%%", percent)
    }

    private func stateLabel(_ state: String) -> String {
        switch state {
        case "successful": return "Funded ✓"
        case "failed": return "Failed"
        case "canceled": return "Canceled"
        default: return "Live"
        }
    }

    private func stateColor(_ state: String) -> Color {
        switch state {
        case "successful": return .green
        case "failed", "canceled": return .red
        default: return .accentColor
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
                .font(.system(size: 14))
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
