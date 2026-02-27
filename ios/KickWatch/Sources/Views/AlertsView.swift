import SwiftUI

struct AlertsView: View {
    @State private var vm = AlertsViewModel()
    @State private var showNewAlert = false

    var body: some View {
        NavigationStack {
            Group {
                if vm.isLoading && vm.alerts.isEmpty {
                    ProgressView()
                } else if vm.alerts.isEmpty {
                    emptyState
                } else {
                    alertList
                }
            }
            .navigationTitle("Alerts")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { showNewAlert = true } label: { Image(systemName: "plus") }
                }
            }
            .sheet(isPresented: $showNewAlert) { NewAlertSheet(vm: vm) }
            .task {
                if let deviceID = NotificationService.shared.deviceID {
                    await vm.load(deviceID: deviceID)
                }
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 16) {
            Image(systemName: "bell.slash").font(.system(size: 48)).foregroundStyle(.secondary)
            Text("No alerts yet").font(.headline)
            Text("Create a keyword alert to get notified when matching campaigns launch.")
                .font(.subheadline).foregroundStyle(.secondary).multilineTextAlignment(.center)
            Button("Create Alert") { showNewAlert = true }
                .buttonStyle(.borderedProminent)
        }
        .padding()
    }

    private var alertList: some View {
        List {
            ForEach(vm.alerts, id: \.id) { alert in
                NavigationLink(destination: AlertMatchesView(alert: alert)) {
                    AlertRowView(alert: alert, vm: vm)
                }
            }
            .onDelete { offsets in
                let toDelete = offsets.map { vm.alerts[$0] }
                for alert in toDelete { Task { await vm.deleteAlert(alert) } }
            }
        }
    }
}

struct AlertRowView: View {
    let alert: AlertDTO
    let vm: AlertsViewModel

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 4) {
                if alert.alert_type == "momentum" {
                    Text("🔥 Momentum Alert").font(.subheadline).fontWeight(.semibold)
                    Text("+\(Int(alert.velocity_thresh ?? 0))% in 24h").font(.caption).foregroundStyle(.secondary)
                } else {
                    Text("\"\(alert.keyword)\"").font(.subheadline).fontWeight(.semibold)
                    Group {
                        if let cat = alert.category_id { Text("Category: \(cat)") }
                        if alert.min_percent > 0 { Text("Min \(Int(alert.min_percent))% funded") }
                    }
                    .font(.caption).foregroundStyle(.secondary)
                }
            }
            Spacer()
            Toggle("", isOn: Binding(
                get: { alert.is_enabled },
                set: { _ in Task { await vm.toggleAlert(alert) } }
            ))
            .labelsHidden()
        }
        .padding(.vertical, 4)
    }
}

struct NewAlertSheet: View {
    let vm: AlertsViewModel
    @State private var alertType = "keyword"
    @State private var keyword = ""
    @State private var minPercent = 0.0
    @State private var velocityThresh = 50.0
    @Environment(\.dismiss) private var dismiss

    private var isValid: Bool {
        guard let deviceID = NotificationService.shared.deviceID, !deviceID.isEmpty else { return false }
        return alertType == "momentum" ? true : !keyword.isEmpty
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Alert Type") {
                    Picker("Type", selection: $alertType) {
                        Text("Keyword").tag("keyword")
                        Text("🔥 Momentum").tag("momentum")
                    }
                    .pickerStyle(.segmented)
                }

                if alertType == "keyword" {
                    Section("Keyword") {
                        TextField("e.g. mechanical keyboard", text: $keyword)
                    }
                    Section("Min % Funded") {
                        Slider(value: $minPercent, in: 0...100, step: 10)
                        Text("\(Int(minPercent))% minimum").foregroundStyle(.secondary)
                    }
                } else {
                    Section {
                        Slider(value: $velocityThresh, in: 25...500, step: 25)
                        Text("Alert when a campaign grows +\(Int(velocityThresh))% in 24h")
                            .foregroundStyle(.secondary)
                    } header: {
                        Text("Growth Threshold")
                    }
                }
            }
            .navigationTitle("New Alert")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        guard let deviceID = NotificationService.shared.deviceID else { return }
                        Task {
                            await vm.createAlert(
                                deviceID: deviceID,
                                alertType: alertType,
                                keyword: keyword,
                                categoryID: nil,
                                minPercent: minPercent,
                                velocityThresh: alertType == "momentum" ? velocityThresh : 0
                            )
                            dismiss()
                        }
                    }
                    .disabled(!isValid)
                }
            }
        }
    }
}

struct AlertMatchesView: View {
    let alert: AlertDTO
    @State private var campaigns: [CampaignDTO] = []
    @State private var isLoading = false

    var body: some View {
        Group {
            if isLoading {
                ProgressView()
            } else if campaigns.isEmpty {
                Text("No matches yet").foregroundStyle(.secondary)
            } else {
                List(campaigns, id: \.pid) { campaign in
                    NavigationLink(destination: CampaignDetailView(campaign: campaign)) {
                        CampaignRowView(campaign: campaign)
                    }
                    .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 16))
                }
                .listStyle(.plain)
            }
        }
        .navigationTitle("\"\(alert.keyword)\" matches")
        .task {
            isLoading = true
            campaigns = (try? await APIClient.shared.fetchAlertMatches(alertID: alert.id)) ?? []
            isLoading = false
        }
    }
}
