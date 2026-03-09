import { apiClient } from './client'
import type { NotificationPreferences } from '../types/api'

export const notificationsAPI = {
  getPreferences: () =>
    apiClient.get<NotificationPreferences>('/notifications/preferences'),

  updatePreferences: (prefs: Partial<NotificationPreferences>) =>
    apiClient.put('/notifications/preferences', prefs),
}
