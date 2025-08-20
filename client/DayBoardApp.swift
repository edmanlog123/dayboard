import SwiftUI
import Combine
import UserNotifications

/// DayBoardApp is the entry point for the SwiftUI application. It sets up
/// a menu bar extra on macOS and an application scene on iOS. The menu
/// bar extra displays the next event, commute estimate, upcoming bills,
/// and pay outlook using data fetched from the backend.
@main
struct DayBoardApp: App {
    @StateObject private var viewModel = DayBoardViewModel()

    var body: some Scene {
        #if os(macOS)
        MenuBarExtra("DayBoard", systemImage: "calendar.badge.clock") {
            ContentView(viewModel: viewModel)
                .onAppear { viewModel.refresh() }
        }
        .menuBarExtraStyle(.window)
        #else
        WindowGroup {
            MainView(viewModel: viewModel)
                .onAppear {
                    UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound]) { _, _ in }
                }
        }
        #endif
    }
}

/// ContentView lays out the user interface for the DayBoard menu bar and
/// application window. It displays the next meeting with a join button,
/// commute estimate, upcoming bills, and pay outlook. Buttons open
/// corresponding screens (not yet implemented).
struct ContentView: View {
    @ObservedObject var viewModel: DayBoardViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let next = viewModel.nextEvent {
                VStack(alignment: .leading, spacing: 2) {
                    Text(next.title)
                        .font(.headline)
                    Text(next.start, style: .time)
                        .font(.subheadline)
                    if let url = next.joinURL {
                        Button("Join") {
                            #if os(macOS)
                            NSWorkspace.shared.open(url)
                            #endif
                        }
                    }
                }
            } else {
                Text("No more events today")
            }

            Divider()

            HStack {
                Text("Commute:")
                Spacer()
                Text(viewModel.commuteCost)
            }

            HStack {
                Text("Bills this week:")
                Spacer()
                Text(viewModel.billsThisWeek)
            }

            HStack {
                Text("Pay outlook:")
                Spacer()
                Text(viewModel.payOutlook)
            }

            Divider()

            Button("Open DayBoard") {
                viewModel.openMainWindow()
            }
        }
        .padding(10)
        .frame(maxWidth: 300)
    }
}

#if os(iOS)
/// MainView provides a tabbed interface on iOS.
struct MainView: View {
    @ObservedObject var viewModel: DayBoardViewModel

    var body: some View {
        TabView {
            TodayView(viewModel: viewModel)
                .tabItem {
                    Label("Today", systemImage: "calendar")
                }
            SubscriptionsView(viewModel: viewModel)
                .tabItem {
                    Label("Subscriptions", systemImage: "creditcard")
                }
            FinancesView(viewModel: viewModel)
                .tabItem {
                    Label("Finances", systemImage: "chart.bar")
                }
            DocumentsView(viewModel: viewModel)
                .tabItem {
                    Label("Documents", systemImage: "doc.text")
                }
            CampusView(viewModel: viewModel)
                .tabItem {
                    Label("Campus", systemImage: "building.2")
                }
        }
        .onAppear { viewModel.refresh() }
    }
}

struct TodayView: View {
    @ObservedObject var viewModel: DayBoardViewModel

    var body: some View {
        NavigationView {
            List {
                Section(header: Text("Next Meeting")) {
                    if let next = viewModel.nextEvent {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(next.title).font(.headline)
                            Text(next.start, style: .time).foregroundStyle(.secondary)
                            if let url = next.joinURL {
                                Button("Join call") {
                                    UIApplication.shared.open(url)
                                }
                                .buttonStyle(.borderedProminent)
                            }
                        }
                    } else {
                        Text("No more events today")
                    }
                }

                Section(header: Text("Today's Burn")) {
                    HStack {
                        Text("Total spent today")
                        Spacer()
                        Text(viewModel.todaysBurn).foregroundStyle(.red)
                    }
                    Button("Add commute cost") {
                        viewModel.showAddCommuteSheet = true
                    }
                }

                Section(header: Text("Email")) {
                    HStack {
                        Text("Unread messages")
                        Spacer()
                        Text("\(viewModel.emailSummary.unreadCount)")
                    }
                    ForEach(viewModel.emailSummary.topSubjects, id: \.self) { subject in
                        Text(subject).font(.caption).foregroundStyle(.secondary)
                    }
                }

                Section(header: Text("Commute")) {
                    HStack {
                        Text("Estimate")
                        Spacer()
                        Text(viewModel.commuteCost)
                    }
                }

                Section(header: Text("Bills This Week")) {
                    HStack {
                        Text("Total due")
                        Spacer()
                        Text(viewModel.billsThisWeek)
                    }
                }

                Section(header: Text("Pay Outlook")) {
                    HStack {
                        Text("Per paycheck")
                        Spacer()
                        Text(viewModel.payOutlook)
                    }
                }
            }
            .navigationTitle("DayBoard")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                    .accessibilityLabel("Refresh")
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.showAddEventSheet = true }) {
                        Image(systemName: "plus")
                    }
                }
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(action: { viewModel.showProfileSheet = true }) {
                        Image(systemName: "person.circle")
                    }
                }
            }
            .refreshable { viewModel.refresh() }
            .sheet(isPresented: $viewModel.showAddEventSheet) {
                AddEventSheet(viewModel: viewModel)
            }
            .sheet(isPresented: $viewModel.showAddCommuteSheet) {
                AddCommuteSheet(viewModel: viewModel)
            }
            .sheet(isPresented: $viewModel.showProfileSheet) {
                ProfileSheet(viewModel: viewModel)
            }
        }
    }
}

struct SubscriptionsView: View {
    @ObservedObject var viewModel: DayBoardViewModel

    var body: some View {
        NavigationView {
            List {
                ForEach(viewModel.subscriptions) { sub in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(sub.merchant).font(.headline)
                            if let due = sub.nextDue {
                                Text("Due: \(due, style: .date)").font(.caption).foregroundStyle(.secondary)
                            }
                        }
                        Spacer()
                        Text(viewModel.centsToDollarString(sub.amountCents))
                            .font(.body)
                    }
                }
                .onDelete(perform: viewModel.deleteSubscriptions)
            }
            .navigationTitle("Subscriptions")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.fetchSubscriptions() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.showAddSubSheet = true }) {
                        Image(systemName: "plus")
                    }
                }
            }
            .refreshable { viewModel.fetchSubscriptions() }
            .sheet(isPresented: $viewModel.showAddSubSheet) {
                AddSubscriptionSheet(viewModel: viewModel)
            }
        }
    }
}

struct FinancesView: View {
    @ObservedObject var viewModel: DayBoardViewModel

    var body: some View {
        NavigationView {
            List {
                Section(header: Text("Today's Summary")) {
                    HStack { Text("Today's burn"); Spacer(); Text(viewModel.todaysBurn).foregroundStyle(.red) }
                    HStack { Text("Per paycheck net"); Spacer(); Text(viewModel.payOutlook) }
                }
                
                Section(header: Text("Weekly Overview")) {
                    HStack { Text("Bills this week"); Spacer(); Text(viewModel.billsThisWeek) }
                    HStack { Text("Commute estimate"); Spacer(); Text(viewModel.commuteCost) }
                }
                
                Section(header: Text("State Tax Comparison")) {
                    ForEach(viewModel.stateTaxComparison, id: \.state) { comparison in
                        VStack(alignment: .leading, spacing: 2) {
                            HStack {
                                Text(comparison.state).font(.headline)
                                Spacer()
                                Text(viewModel.centsToDollarString(comparison.netPayCents))
                            }
                            Text("Tax rate: \(comparison.taxRate, specifier: "%.1f")%")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
                
                Section(header: Text("Housing Cost Comparison")) {
                    ForEach(viewModel.housingComparison, id: \.city) { housing in
                        HStack {
                            VStack(alignment: .leading) {
                                Text(housing.city).font(.headline)
                                Text("Avg rent: \(viewModel.centsToDollarString(housing.avgRentCents))")
                                    .font(.caption).foregroundStyle(.secondary)
                            }
                            Spacer()
                            Text("Net after rent: \(viewModel.centsToDollarString(housing.netAfterRentCents))")
                        }
                    }
                }
            }
            .navigationTitle("Finances")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.fetchFinanceData() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .refreshable { viewModel.fetchFinanceData() }
        }
    }
}
#endif

/// DayBoardViewModel orchestrates data fetching from the backend and
/// transforms it into view-friendly formats. It uses Combine to
/// asynchronously load today's agenda, commute estimate, subscriptions,
/// and pay outlook. Real network requests should be implemented here.
final class DayBoardViewModel: ObservableObject {
    // Published properties drive UI updates.
    @Published var nextEvent: DayBoardEvent?
    @Published var commuteCost: String = "–"
    @Published var billsThisWeek: String = "–"
    @Published var payOutlook: String = "–"
    @Published var todaysBurn: String = "–"
    @Published var emailSummary: EmailSummary = EmailSummary(unreadCount: 0, topSubjects: [])
    @Published var stateTaxComparison: [StateTaxComparison] = []
    @Published var housingComparison: [HousingComparison] = []
    @Published var campusEvents: [CampusEvent] = []
    @Published var aiAdvice: String = ""
    #if os(iOS)
    @Published var subscriptions: [SubscriptionItem] = []
    @Published var showAddSubSheet: Bool = false
    @Published var showAddEventSheet: Bool = false
    @Published var showAddCommuteSheet: Bool = false
    @Published var showProfileSheet: Bool = false
    @Published var showDocumentScanner: Bool = false
    @Published var currentProfile: BackendProfile?
    @Published var scannedDocuments: [ScannedDocument] = []
    #endif

    private var cancellables = Set<AnyCancellable>()

    /// The base URL of the DayBoard backend. Replace this with your deployed
    /// backend URL when running in production. When testing locally you can
    /// leave the default value (assuming the Go server runs on port 8080).
    private let baseURL = URL(string: "http://localhost:8080/api/v1")!

    /// A hard-coded user ID for demonstration purposes. In a production app,
    /// you would generate this after user login and persist it (e.g. in the
    /// keychain). The Go backend expects a `user_id` query parameter on
    /// certain endpoints. Replace this with the UUID of the logged-in user.
    private let userID = "00000000-0000-0000-0000-000000000000"

    /// Refresh reloads all data required for the menu bar display.
    func refresh() {
        fetchAgenda()
        fetchSubscriptions()
        fetchCommuteEstimate()
        fetchPayOutlook()
        fetchTodaysBurn()
        fetchEmailSummary()
        fetchFinanceData()
        fetchCampusEvents()
    }
    
    func fetchFinanceData() {
        fetchStateTaxComparison()
        fetchHousingComparison()
    }

    /// openMainWindow would present the full application window. Stubbed for now.
    func openMainWindow() {
        // TODO: Implement main window presentation.
    }

    // MARK: - Private network calls

    private func fetchAgenda() {
        let url = baseURL.appendingPathComponent("agenda/today")
        var comps = URLComponents(url: url, resolvingAgainstBaseURL: false)!
        comps.queryItems = [URLQueryItem(name: "user_id", value: userID)]
        guard let finalURL = comps.url else { return }
        URLSession.shared.dataTaskPublisher(for: finalURL)
            .tryMap { data, response -> [DayBoardEvent] in
                let decoder = JSONDecoder()
                decoder.dateDecodingStrategy = .iso8601
                let items = try decoder.decode([BackendEvent].self, from: data)
                return items.map { DayBoardEvent(id: $0.id.uuidString, title: $0.title, start: $0.start, joinURL: URL(string: $0.joinURL ?? "")) }
            }
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] events in
                self?.nextEvent = events.first
            })
            .store(in: &cancellables)
    }

    func fetchSubscriptions() {
        let url = baseURL.appendingPathComponent("subs")
        var comps = URLComponents(url: url, resolvingAgainstBaseURL: false)!
        comps.queryItems = [URLQueryItem(name: "user_id", value: userID)]
        guard let finalURL = comps.url else { return }
        URLSession.shared.dataTaskPublisher(for: finalURL)
            .tryMap { data, response -> [BackendSubscription] in
                let decoder = JSONDecoder()
                decoder.dateDecodingStrategy = .iso8601
                return try decoder.decode([BackendSubscription].self, from: data)
            }
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] subs in
                // Compute bills due in the next 7 days.
                let now = Date()
                let weekAhead = Calendar.current.date(byAdding: .day, value: 7, to: now) ?? now
                var totalCents = 0
                for sub in subs {
                    if let due = sub.nextDue {
                        if due <= weekAhead {
                            totalCents += sub.amountCents
                        }
                    }
                }
                self?.billsThisWeek = self?.centsToDollarString(totalCents) ?? "–"
                #if os(iOS)
                self?.subscriptions = subs.map { SubscriptionItem(_id: $0.id, merchant: $0.merchant, amountCents: $0.amountCents, nextDue: $0.nextDue) }
                #endif
            })
            .store(in: &cancellables)
    }

    private func fetchCommuteEstimate() {
        // For demonstration we need to know the user's home and office addresses. In
        // production you'd fetch the profile first. Here we call the profile endpoint.
        let profileURL = baseURL.appendingPathComponent("profile")
        var comps = URLComponents(url: profileURL, resolvingAgainstBaseURL: false)!
        comps.queryItems = [URLQueryItem(name: "user_id", value: userID)]
        guard let profileFinalURL = comps.url else { return }
        URLSession.shared.dataTaskPublisher(for: profileFinalURL)
            .map { data, _ -> BackendProfile? in
                let decoder = JSONDecoder()
                decoder.dateDecodingStrategy = .iso8601
                // Empty profile may return {} which decodes to nil if we use optional.
                return try? decoder.decode(BackendProfile.self, from: data)
            }
            .catch { _ in Just(nil) }
            .flatMap { [weak self] profile -> AnyPublisher<commuteEstResponse?, Never> in
                #if os(iOS)
                self?.currentProfile = profile
                #endif
                guard let profile = profile, let self = self else { return Just(nil).eraseToAnyPublisher() }
                // Compose commute API call
                let commuteURL = self.baseURL.appendingPathComponent("commute/estimate")
                var comps = URLComponents(url: commuteURL, resolvingAgainstBaseURL: false)!
                comps.queryItems = [
                    URLQueryItem(name: "from", value: profile.homeAddr),
                    URLQueryItem(name: "to", value: profile.officeAddr),
                ]
                guard let finalCommuteURL = comps.url else { return Just(nil).eraseToAnyPublisher() }
                return URLSession.shared.dataTaskPublisher(for: finalCommuteURL)
                    .map { $0.data }
                    .decode(type: commuteEstResponse.self, decoder: JSONDecoder())
                    .map(Optional.some)
                    .catch { _ in Just(nil) }
                    .eraseToAnyPublisher()
            }
            .receive(on: DispatchQueue.main)
            .sink(receiveValue: { [weak self] est in
                guard let self = self, let est = est else { return }
                let low = self.centsToDollarString(est.estCostLowCents)
                let high = self.centsToDollarString(est.estCostHighCents)
                self.commuteCost = "\(low)–\(high)"
            })
            .store(in: &cancellables)
    }

    func fetchPayOutlook() {
        // Fetch profile first to compute income and hours. Then call taxes API.
        let profileURL = baseURL.appendingPathComponent("profile")
        var comps = URLComponents(url: profileURL, resolvingAgainstBaseURL: false)!
        comps.queryItems = [URLQueryItem(name: "user_id", value: userID)]
        guard let profileFinalURL = comps.url else { return }
        URLSession.shared.dataTaskPublisher(for: profileFinalURL)
            .map { data, _ -> BackendProfile? in
                let decoder = JSONDecoder()
                decoder.dateDecodingStrategy = .iso8601
                return try? decoder.decode(BackendProfile.self, from: data)
            }
            .catch { _ in Just(nil) }
            .flatMap { [weak self] profile -> AnyPublisher<TaxResult?, Never> in
                #if os(iOS)
                self?.currentProfile = profile
                #endif
                guard let self = self, let profile = profile else { return Just(nil).eraseToAnyPublisher() }
                // Derive weekly income in cents.
                var incomeCents = 0
                if let hourly = profile.hourlyCents, let hours = profile.hoursPerWeek {
                    incomeCents = hourly * hours * 52 // annual income
                } else if let stipend = profile.stipendCents {
                    incomeCents = stipend
                }
                let termWeeks = 12 // Example 12-week internship for projection
                let body: [String: Any] = [
                    "incomeCents": incomeCents,
                    "state": profile.state,
                    "filingStatus": "single",
                    "payFreq": profile.payFreq ?? "biweekly",
                    "termWeeks": termWeeks,
                ]
                guard let url = self.baseURL.appendingPathComponent("estimate/taxes") as URL? else {
                    return Just(nil).eraseToAnyPublisher()
                }
                var req = URLRequest(url: url)
                req.httpMethod = "POST"
                req.setValue("application/json", forHTTPHeaderField: "Content-Type")
                req.httpBody = try? JSONSerialization.data(withJSONObject: body)
                return URLSession.shared.dataTaskPublisher(for: req)
                    .map { $0.data }
                    .decode(type: TaxResult.self, decoder: JSONDecoder())
                    .map(Optional.some)
                    .catch { _ in Just(nil) }
                    .eraseToAnyPublisher()
            }
            .receive(on: DispatchQueue.main)
            .sink(receiveValue: { [weak self] res in
                guard let res = res else { return }
                self?.payOutlook = self?.centsToDollarString(res.perPaycheckNetCents) ?? "$0.00"
            })
            .store(in: &cancellables)
    }

    #if os(iOS)
    // Add a subscription via demo API and refresh list.
    func addSubscription(merchant: String, amountCents: Int, cadenceDays: Int, nextDue: Date?) {
        let url = baseURL.appendingPathComponent("subs")
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        var body: [String: Any] = [
            "merchant": merchant,
            "amountCents": amountCents,
            "cadenceDays": cadenceDays,
        ]
        if let next = nextDue {
            let iso = ISO8601DateFormatter().string(from: next)
            body["nextDue"] = iso
        }
        req.httpBody = try? JSONSerialization.data(withJSONObject: body)
        URLSession.shared.dataTask(with: req) { _, _, _ in
            DispatchQueue.main.async { self.fetchSubscriptions() }
        }.resume()
    }

    func deleteSubscriptions(at offsets: IndexSet) {
        let ids = offsets.map { subscriptions[$0]._id }
        // Optimistically update UI
        subscriptions.remove(atOffsets: offsets)
        // Call demo delete endpoint (no real persistence in demo)
        for id in ids {
            let endpoint = baseURL.appendingPathComponent("subs/\(id.uuidString)")
            var req = URLRequest(url: endpoint)
            req.httpMethod = "DELETE"
            URLSession.shared.dataTask(with: req).resume()
        }
    }

    func addEvent(title: String, start: Date, end: Date, joinURL: String) {
        let url = baseURL.appendingPathComponent("agenda/today")
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: Any] = [
            "title": title,
            "start": ISO8601DateFormatter().string(from: start),
            "end": ISO8601DateFormatter().string(from: end),
            "joinURL": joinURL
        ]
        req.httpBody = try? JSONSerialization.data(withJSONObject: body)
        URLSession.shared.dataTask(with: req) { _, _, _ in
            DispatchQueue.main.async { 
                self.fetchAgenda()
                self.scheduleNotificationForNextEvent()
            }
        }.resume()
    }

    func fetchTodaysBurn() {
        let url = baseURL.appendingPathComponent("daily/burn")
        URLSession.shared.dataTaskPublisher(for: url)
            .map { $0.data }
            .decode(type: DailyBurnResponse.self, decoder: JSONDecoder())
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] burn in
                self?.todaysBurn = self?.centsToDollarString(burn.totalCents) ?? "–"
            })
            .store(in: &cancellables)
    }

    func fetchEmailSummary() {
        let url = baseURL.appendingPathComponent("email/summary")
        URLSession.shared.dataTaskPublisher(for: url)
            .map { $0.data }
            .decode(type: EmailSummary.self, decoder: JSONDecoder())
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] summary in
                self?.emailSummary = summary
            })
            .store(in: &cancellables)
    }

    func addCommuteEntry(from: String, to: String, costCents: Int, method: String) {
        let url = baseURL.appendingPathComponent("commute/entries")
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: Any] = [
            "from": from,
            "to": to,
            "costCents": costCents,
            "method": method
        ]
        req.httpBody = try? JSONSerialization.data(withJSONObject: body)
        URLSession.shared.dataTask(with: req) { _, _, _ in
            DispatchQueue.main.async { 
                self.fetchTodaysBurn()
                self.fetchCommuteEstimate()
            }
        }.resume()
    }

    func scheduleNotificationForNextEvent() {
        guard let next = nextEvent else { return }
        let content = UNMutableNotificationContent()
        content.title = "Upcoming Meeting"
        content.body = "\(next.title) starts in 10 minutes"
        content.sound = .default
        
        let tenMinutesBefore = next.start.addingTimeInterval(-600)
        if tenMinutesBefore > Date() {
            let trigger = UNTimeIntervalNotificationTrigger(timeInterval: tenMinutesBefore.timeIntervalSinceNow, repeats: false)
            let request = UNNotificationRequest(identifier: "meeting-\(next.id)", content: content, trigger: trigger)
            UNUserNotificationCenter.current().add(request)
        }
    }

    func updateProfile(homeAddr: String, officeAddr: String, hourlyPay: Int, hoursPerWeek: Int, state: String, foodCost: Int) {
        let url = baseURL.appendingPathComponent("profile")
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: Any] = [
            "homeAddr": homeAddr,
            "officeAddr": officeAddr,
            "hourlyCents": hourlyPay,
            "hoursPerWeek": hoursPerWeek,
            "state": state,
            "foodCostCents": foodCost,
            "payFreq": "biweekly",
            "inOfficeDays": 3
        ]
        req.httpBody = try? JSONSerialization.data(withJSONObject: body)
        URLSession.shared.dataTask(with: req) { _, _, _ in
            DispatchQueue.main.async { 
                self.refresh()
            }
        }.resume()
    }

    func fetchStateTaxComparison() {
        let url = baseURL.appendingPathComponent("finance/state-comparison")
        URLSession.shared.dataTaskPublisher(for: url)
            .map { $0.data }
            .decode(type: [StateTaxComparison].self, decoder: JSONDecoder())
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] comparisons in
                self?.stateTaxComparison = comparisons
            })
            .store(in: &cancellables)
    }

    func fetchHousingComparison() {
        let url = baseURL.appendingPathComponent("finance/housing-comparison")
        URLSession.shared.dataTaskPublisher(for: url)
            .map { $0.data }
            .decode(type: [HousingComparison].self, decoder: JSONDecoder())
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] housing in
                self?.housingComparison = housing
            })
            .store(in: &cancellables)
    }

    func fetchCampusEvents() {
        let url = baseURL.appendingPathComponent("campus/events")
        URLSession.shared.dataTaskPublisher(for: url)
            .map { $0.data }
            .decode(type: [CampusEvent].self, decoder: JSONDecoder())
            .receive(on: DispatchQueue.main)
            .sink(receiveCompletion: { _ in }, receiveValue: { [weak self] events in
                self?.campusEvents = events
            })
            .store(in: &cancellables)
    }

    func getAIAdvice(query: String) {
        let url = baseURL.appendingPathComponent("ai/advice")
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body = ["query": query]
        req.httpBody = try? JSONSerialization.data(withJSONObject: body)
        URLSession.shared.dataTask(with: req) { data, _, _ in
            guard let data = data,
                  let response = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let advice = response["advice"] as? String else { return }
            DispatchQueue.main.async {
                self.aiAdvice = advice
            }
        }.resume()
    }
    #endif

    // Helper to convert cents to a dollar string like "$12.34".
    func centsToDollarString(_ cents: Int) -> String {
        let dollars = Double(cents) / 100.0
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        return formatter.string(from: NSNumber(value: dollars)) ?? "$0.00"
    }
}

// MARK: - Backend models for decoding

/// BackendEvent mirrors the JSON returned by the /agenda/today endpoint.
fileprivate struct BackendEvent: Decodable {
    let id: UUID
    let start: Date
    let end: Date
    let title: String
    let joinURL: String?
    let location: String?
}

/// BackendSubscription mirrors the /subs endpoint.
fileprivate struct BackendSubscription: Decodable {
    let id: UUID
    let merchant: String
    let amountCents: Int
    let cadenceDays: Int
    let nextDue: Date?
    let source: String
    let isActive: Bool
}

/// BackendProfile mirrors the /profile endpoint.
struct BackendProfile: Decodable {
    let homeAddr: String
    let officeAddr: String
    let city: String
    let state: String
    let hourlyCents: Int?
    let hoursPerWeek: Int?
    let stipendCents: Int?
    let payFreq: String?
    let startDate: Date?
    let inOfficeDays: Int
    let foodCostCents: Int
}

/// commuteEstResponse mirrors the response from /commute/estimate.
fileprivate struct commuteEstResponse: Decodable {
    let distanceMiles: Double
    let durationMinutes: Double
    let estCostLowCents: Int
    let estCostHighCents: Int
}

/// TaxResult mirrors the JSON returned by the /estimate/taxes endpoint. Swift's
/// coding keys use camelCase to map to JSON keys returned by the Go backend.
fileprivate struct TaxResult: Decodable {
    let federalCents: Int
    let stateCents: Int
    let ficaCents: Int
    let perPaycheckNetCents: Int
    let termNetCents: Int
}

/// DayBoardEvent represents a calendar event normalized for the view.
struct DayBoardEvent {
    let id: String
    let title: String
    let start: Date
    let joinURL: URL?
}

struct EmailSummary: Codable {
    let unreadCount: Int
    let topSubjects: [String]
}

struct DailyBurnResponse: Codable {
    let totalCents: Int
}

struct StateTaxComparison: Codable {
    let state: String
    let taxRate: Double
    let netPayCents: Int
}

struct HousingComparison: Codable {
    let city: String
    let avgRentCents: Int
    let netAfterRentCents: Int
}

struct CampusEvent: Codable, Identifiable {
    let id: UUID
    let title: String
    let date: Date
    let location: String
    let category: String
}

struct ScannedDocument: Identifiable {
    let id: UUID
    let name: String
    let type: String
    let dateScanned: Date
    let extractedText: String
}

#if os(iOS)
struct SubscriptionItem: Identifiable {
    let _id: UUID
    let merchant: String
    let amountCents: Int
    let nextDue: Date?
    var id: UUID { _id }
}

struct DocumentsView: View {
    @ObservedObject var viewModel: DayBoardViewModel

    var body: some View {
        NavigationView {
            List {
                Section(header: Text("AI Assistant")) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Ask about internships, finances, or career advice")
                            .font(.caption).foregroundStyle(.secondary)
                        if !viewModel.aiAdvice.isEmpty {
                            Text(viewModel.aiAdvice)
                                .padding(8)
                                .background(Color.blue.opacity(0.1))
                                .cornerRadius(8)
                        }
                        HStack {
                            Button("Salary Negotiation") { viewModel.getAIAdvice(query: "How to negotiate internship salary?") }
                            Button("Interview Tips") { viewModel.getAIAdvice(query: "Best internship interview tips") }
                        }
                    }
                }
                
                Section(header: Text("Document Scanner")) {
                    Button("Scan Resume/Document") {
                        viewModel.showDocumentScanner = true
                    }
                    ForEach(viewModel.scannedDocuments) { doc in
                        VStack(alignment: .leading) {
                            Text(doc.name).font(.headline)
                            Text(doc.type).font(.caption).foregroundStyle(.secondary)
                            Text(doc.extractedText.prefix(100) + "...").font(.caption)
                        }
                    }
                }
            }
            .navigationTitle("Documents & AI")
            .sheet(isPresented: $viewModel.showDocumentScanner) {
                DocumentScannerView(viewModel: viewModel)
            }
        }
    }
}

struct CampusView: View {
    @ObservedObject var viewModel: DayBoardViewModel

    var body: some View {
        NavigationView {
            List {
                Section(header: Text("Campus Events")) {
                    ForEach(viewModel.campusEvents) { event in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(event.title).font(.headline)
                            HStack {
                                Text(event.date, style: .date).font(.caption)
                                Spacer()
                                Text(event.category).font(.caption).foregroundStyle(.blue)
                            }
                            Text(event.location).font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
                
                Section(header: Text("Sports & Entertainment")) {
                    Button("Find Local Events") {
                        // TODO: Integrate with SeatGeek or similar API
                    }
                    Text("Coming soon: Local sports tickets, concerts, and campus events")
                        .font(.caption).foregroundStyle(.secondary)
                }
            }
            .navigationTitle("Campus Life")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.fetchCampusEvents() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .refreshable { viewModel.fetchCampusEvents() }
        }
    }
}

struct DocumentScannerView: View {
    @ObservedObject var viewModel: DayBoardViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationView {
            VStack {
                Text("Document Scanner")
                    .font(.title)
                    .padding()
                
                Text("In a real implementation, this would use VisionKit's DocumentCamera to scan documents and extract text using OCR.")
                    .multilineTextAlignment(.center)
                    .padding()
                
                Button("Simulate Scan Resume") {
                    let mockDoc = ScannedDocument(
                        id: UUID(),
                        name: "Resume.pdf",
                        type: "Resume",
                        dateScanned: Date(),
                        extractedText: "John Doe\nSoftware Engineering Student\nSkills: Swift, Go, PostgreSQL\nExperience: iOS Development Intern"
                    )
                    viewModel.scannedDocuments.append(mockDoc)
                    dismiss()
                }
                .buttonStyle(.borderedProminent)
                
                Spacer()
            }
            .navigationTitle("Scanner")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
}
#endif

#if os(iOS)
struct AddSubscriptionSheet: View {
    @ObservedObject var viewModel: DayBoardViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var merchant: String = ""
    @State private var amount: String = ""
    @State private var cadence: String = "30"
    @State private var hasNextDue: Bool = false
    @State private var nextDue: Date = Date().addingTimeInterval(86400)

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Merchant")) {
                    TextField("e.g. Spotify", text: $merchant)
                }
                Section(header: Text("Amount (USD)")) {
                    TextField("9.99", text: $amount).keyboardType(.decimalPad)
                }
                Section(header: Text("Cadence (days)")) {
                    TextField("30", text: $cadence).keyboardType(.numberPad)
                }
                Section(header: Text("Next Due")) {
                    Toggle("Set next due", isOn: $hasNextDue)
                    if hasNextDue {
                        DatePicker("Date", selection: $nextDue, displayedComponents: .date)
                    }
                }
            }
            .navigationTitle("Add Subscription")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        let cents = Self.toCents(amount)
                        let days = Int(cadence) ?? 30
                        viewModel.addSubscription(
                            merchant: merchant.trimmingCharacters(in: .whitespacesAndNewlines),
                            amountCents: cents,
                            cadenceDays: days,
                            nextDue: hasNextDue ? nextDue : nil
                        )
                        dismiss()
                    }.disabled(merchant.isEmpty || Self.toCents(amount) <= 0)
                }
            }
        }
    }

    private static func toCents(_ dollars: String) -> Int {
        let trimmed = dollars.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return 0 }
        let v = Double(trimmed) ?? 0
        return Int((v * 100.0).rounded())
    }
}

struct AddCommuteSheet: View {
    @ObservedObject var viewModel: DayBoardViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var from: String = "Home"
    @State private var to: String = "Office"
    @State private var cost: String = "12.50"
    @State private var method: String = "Uber"

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Route")) {
                    TextField("From", text: $from)
                    TextField("To", text: $to)
                }
                Section(header: Text("Cost (USD)")) {
                    TextField("12.50", text: $cost).keyboardType(.decimalPad)
                }
                Section(header: Text("Method")) {
                    TextField("Uber", text: $method)
                }
            }
            .navigationTitle("Add Commute")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        let cents = Self.toCents(cost)
                        viewModel.addCommuteEntry(from: from, to: to, costCents: cents, method: method)
                        dismiss()
                    }.disabled(from.isEmpty || to.isEmpty || Self.toCents(cost) <= 0)
                }
            }
        }
    }

    private static func toCents(_ dollars: String) -> Int {
        let trimmed = dollars.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return 0 }
        let v = Double(trimmed) ?? 0
        return Int((v * 100.0).rounded())
    }
}

struct ProfileSheet: View {
    @ObservedObject var viewModel: DayBoardViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var homeAddr: String = ""
    @State private var officeAddr: String = ""
    @State private var hourlyPay: String = "25.00"
    @State private var hoursPerWeek: String = "40"
    @State private var state: String = "IN"
    @State private var foodCost: String = "12.00"

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Addresses")) {
                    TextField("Home address", text: $homeAddr)
                    TextField("Office address", text: $officeAddr)
                }
                Section(header: Text("Pay")) {
                    TextField("Hourly rate (USD)", text: $hourlyPay).keyboardType(.decimalPad)
                    TextField("Hours per week", text: $hoursPerWeek).keyboardType(.numberPad)
                    TextField("State (e.g. IN)", text: $state)
                }
                Section(header: Text("Daily Costs")) {
                    TextField("Food cost (USD)", text: $foodCost).keyboardType(.decimalPad)
                }
            }
            .navigationTitle("Profile")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        viewModel.updateProfile(
                            homeAddr: homeAddr,
                            officeAddr: officeAddr,
                            hourlyPay: Self.toCents(hourlyPay),
                            hoursPerWeek: Int(hoursPerWeek) ?? 40,
                            state: state,
                            foodCost: Self.toCents(foodCost)
                        )
                        dismiss()
                    }
                }
            }
        }
        .onAppear {
            // Pre-populate with current profile
            homeAddr = viewModel.currentProfile?.homeAddr ?? ""
            officeAddr = viewModel.currentProfile?.officeAddr ?? ""
            if let hourly = viewModel.currentProfile?.hourlyCents {
                hourlyPay = String(format: "%.2f", Double(hourly) / 100.0)
            }
            if let hours = viewModel.currentProfile?.hoursPerWeek {
                hoursPerWeek = String(hours)
            }
            state = viewModel.currentProfile?.state ?? "IN"
            foodCost = String(format: "%.2f", Double(viewModel.currentProfile?.foodCostCents ?? 1200) / 100.0)
        }
    }

    private static func toCents(_ dollars: String) -> Int {
        let trimmed = dollars.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return 0 }
        let v = Double(trimmed) ?? 0
        return Int((v * 100.0).rounded())
    }
}
#endif

#if os(iOS)
struct AddEventSheet: View {
    @ObservedObject var viewModel: DayBoardViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var title: String = ""
    @State private var start: Date = Date().addingTimeInterval(1800)
    @State private var end: Date = Date().addingTimeInterval(3600)
    @State private var joinURL: String = "https://meet.google.com/demo-room"

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Title")) { TextField("Standup", text: $title) }
                Section(header: Text("Start")) { DatePicker("", selection: $start, displayedComponents: [.date, .hourAndMinute]).labelsHidden() }
                Section(header: Text("End")) { DatePicker("", selection: $end, displayedComponents: [.date, .hourAndMinute]).labelsHidden() }
                Section(header: Text("Join URL")) { TextField("https://…", text: $joinURL).keyboardType(.URL) }
            }
            .navigationTitle("Add Event")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        viewModel.addEvent(title: title, start: start, end: end, joinURL: joinURL)
                        dismiss()
                    }.disabled(title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || end <= start)
                }
            }
        }
    }
}
#endif


