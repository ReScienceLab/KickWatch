import SwiftUI
import SwiftData

struct DiscoverView: View {
    @State private var vm = DiscoverViewModel()
    @State private var searchText = ""
    @State private var showSearch = false

    private let sortOptions = [("hot", "🔥 Hot"), ("trending", "Trending"), ("newest", "New"), ("ending", "Ending")]

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                sortPicker
                categoryScroll
                campaignList
            }
            .navigationTitle("Discover")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { showSearch = true } label: { Image(systemName: "magnifyingglass") }
                }
            }
            .sheet(isPresented: $showSearch) { SearchView() }
            .task { await vm.loadCategories(); await vm.load() }
            .refreshable { await vm.load() }
        }
    }

    private var sortPicker: some View {
        Picker("Sort", selection: Binding(
            get: { vm.selectedSort },
            set: { newSort in Task { await vm.selectSort(newSort) } }
        )) {
            ForEach(sortOptions, id: \.0) { key, label in Text(label).tag(key) }
        }
        .pickerStyle(.segmented)
        .padding(.horizontal)
        .padding(.vertical, 8)
        .disabled(vm.isLoading && !vm.campaigns.isEmpty)
        .opacity(vm.isLoading && !vm.campaigns.isEmpty ? 0.6 : 1.0)
    }

    private var categoryScroll: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                CategoryChip(title: "All", isSelected: vm.selectedCategoryID == nil) {
                    Task { await vm.selectCategory(nil) }
                }
                ForEach(vm.categories.filter { $0.parent_id == nil }, id: \.id) { cat in
                    CategoryChip(title: cat.name, isSelected: vm.selectedCategoryID == cat.id) {
                        Task { await vm.selectCategory(cat.id) }
                    }
                }
            }
            .padding(.horizontal)
        }
        .padding(.bottom, 4)
        .disabled(vm.isLoading && !vm.campaigns.isEmpty)
        .opacity(vm.isLoading && !vm.campaigns.isEmpty ? 0.6 : 1.0)
    }

    private var campaignList: some View {
        Group {
            if vm.isLoading && vm.campaigns.isEmpty {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let err = vm.error {
                Text(err).foregroundStyle(.secondary).padding()
            } else {
                ZStack {
                    List {
                        ForEach(vm.campaigns, id: \.pid) { campaign in
                            NavigationLink(destination: CampaignDetailView(campaign: campaign)) {
                                CampaignRowView(campaign: campaign)
                            }
                            .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 16))
                            .onAppear {
                                if campaign.pid == vm.campaigns.last?.pid {
                                    Task { await vm.loadMore() }
                                }
                            }
                        }
                        if vm.isLoadingMore {
                            ProgressView().frame(maxWidth: .infinity)
                        }
                    }
                    .listStyle(.plain)
                    .opacity(vm.isLoading && !vm.campaigns.isEmpty ? 0.3 : 1.0)
                    
                    if vm.isLoading && !vm.campaigns.isEmpty {
                        ProgressView()
                            .scaleEffect(1.5)
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                            .background(Color(uiColor: .systemBackground).opacity(0.5))
                    }
                }
            }
        }
    }
}
