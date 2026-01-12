// Package models defines the core domain models for Splitwiser.
//
// # Current Models (MVP)
//
// The following models are actively used in the MVP:
//   - Bill: Represents a bill to be split among participants
//   - Item: Individual line items on a bill
//   - PersonSplit: Calculated split result for one person
//
// For the MVP, participants are identified by name strings (no user accounts).
//
// # Future Models
//
// The following models are defined but not yet used (for future auth features):
//   - User: Registered user account (requires authentication)
//   - Group: Recurring group of people who frequently split bills
//
// # Design Principles
//
// 1. **MVP simplicity**: Bills use participant names (strings) for now
// 2. **Future extensibility**: Models designed to add user IDs later without breaking changes
// 3. **Avoid circular references**: Use ID strings instead of pointers for relationships
// 4. **Clear documentation**: Each model documents its current state and future plans
//
// # Migration Path
//
// When adding authentication:
//   1. Add User table to database
//   2. Add optional CreatorID to Bill model
//   3. Participant names remain strings for backward compatibility
//   4. New bills can optionally link to User IDs
//   5. Add Group support for recurring participants
package models
