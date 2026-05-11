## ADDED Requirements

### Requirement: Bounded metrics enrichment
Metrics enrichment SHALL respect caller cancellation and bounded network execution so optional provider calls cannot indefinitely delay item sync.

#### Scenario: Metrics provider exceeds timeout or cancellation
- **WHEN** a metrics provider call does not complete before its context is cancelled or its timeout is reached
- **THEN** metrics enrichment for that item stops
- **AND** the item sync remains successful if the item content and runtime metadata were saved successfully

#### Scenario: Metrics timeout does not mark source failed
- **WHEN** a matched metrics provider times out while processing an item
- **THEN** the source sync is not marked failed solely because metrics were unavailable
- **AND** the item remains available without metrics

#### Scenario: Metrics provider receives item sync context
- **WHEN** sync asks a metrics provider to fetch metrics for an item
- **THEN** the provider receives a context derived from the source sync context
