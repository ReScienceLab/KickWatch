## UI component best practices

- **Single Responsibility**: Each SwiftUI view should have one clear purpose and do it well
- **Reusability**: Design views to be reused across different contexts with configurable parameters
- **Composability**: Build complex UIs by combining smaller, simpler views rather than monolithic structures
- **Clear Interface**: Define explicit parameters with sensible defaults for ease of use
- **Encapsulation**: Keep internal implementation details private and expose only necessary bindings
- **Consistent Naming**: Use clear, descriptive names that indicate the view's purpose (e.g., CampaignRow, SparklineView)
- **State Management**: Use @State for local view state, @Query for SwiftData, @Environment for shared data
- **Minimal Props**: Keep the number of parameters manageable; if a view needs many parameters, consider composition or splitting it
- **Documentation**: Document view usage and parameters with code comments for easier adoption by team members
