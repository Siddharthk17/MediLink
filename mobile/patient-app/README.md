# MediLink Patient Mobile App

> **Status: Planned — Not Yet Implemented**

This directory is reserved for a future React Native mobile application for patients. The mobile app is not part of the current release.

## Current Patient Access

Patients currently use the **Patient Dashboard** — a full-featured web application accessible at `http://localhost:8180/patient/` when running via Docker Compose. The web dashboard provides:

- Health records (labs, medications, conditions, allergies, immunizations)
- Consent management with access audit log
- Document viewing
- Patient timeline
- Search across all health data
- Notification preferences
- Profile and password management

## Future Plans

When implemented, the mobile app will use:

| Technology | Purpose |
|---|---|
| React Native + Expo | Cross-platform mobile framework |
| Firebase Cloud Messaging | Push notifications |
| `@medilink/shared` | Shared TypeScript types and API clients |

The backend already supports mobile-specific features:
- FCM token registration (`POST /notifications/fcm-token`)
- FCM token revocation (`DELETE /notifications/fcm-token`)
- All REST APIs work with any HTTP client

## Related

- [Patient Dashboard](../../frontend/patient-dashboard/) — current web-based patient interface
- [API Reference](../../docs/API.md) — backend API documentation
- [Architecture](../../docs/ARCHITECTURE.md) — system architecture overview
