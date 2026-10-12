/**
 * Public surface of the settings feature. App routes and other features import
 * from here only — internals (mock-db, api) stay private, except the
 * synchronous `readMonthStartDay` reader that the budget api needs to build its
 * period range (mirrors how budget imports `readToken` from auth).
 */

export {
  CURRENCIES,
  DEFAULT_SETTINGS,
  type Currency,
  type Settings,
  type UpdateSettingsInput,
} from "./schemas"

export { settingsKeys, useSettings, useToday } from "./queries"

export { useUpdateSettings } from "./mutations"

export { isHistoryResetRequired, readMonthStartDay, readToday } from "./api"
