import type { ChannelKind } from '../api';

// FORMAT_LABEL names each receiver the way its own product does, because a
// dropdown of slugs makes the operator guess which one their URL belongs to.
export const FORMAT_LABEL: Record<string, string> = {
  auto: 'Detect from URL',
  discord: 'Discord',
  slack: 'Slack',
  mattermost: 'Mattermost',
  rocketchat: 'Rocket.Chat',
  googlechat: 'Google Chat',
  teams: 'Microsoft Teams (connector)',
  teamsflow: 'Microsoft Teams (Power Automate)',
  webex: 'Webex',
  ntfy: 'ntfy',
  gotify: 'Gotify',
  telegram: 'Telegram',
  pagerduty: 'PagerDuty',
  generic: "Raw JSON (your own sink)",
  custom: 'Custom template',
};

// EVENT_LABEL says what each event type means, because a checkbox list of slugs
// makes the operator guess what actually fires. A type with no entry here still
// renders, under its own name.
export const EVENT_LABEL: Record<string, string> = {
  down: 'Service down',
  log_error: 'Service logging errors',
  agent_offline: 'Agent offline',
  cpu_high: 'CPU over threshold',
  mem_high: 'Memory over threshold',
  disk_high: 'Disk over threshold',
  temp_high: 'Temperature over threshold',
  load_high: 'Load over threshold',
  net_high: 'Network over threshold',
  access_request: 'Credential request',
};

// extraConfigKey names the one config field a receiver needs in the body that no
// URL can supply. Telegram will not post without a chat, and PagerDuty routes on
// an integration key rather than the path.
export function extraConfigKey(format: ChannelKind): 'chat_id' | 'routing_key' | null {
  if (format === 'telegram') {
    return 'chat_id';
  }
  if (format === 'pagerduty') {
    return 'routing_key';
  }
  return null;
}

export function formatNeedsTemplate(format: ChannelKind): boolean {
  return format === 'custom';
}
