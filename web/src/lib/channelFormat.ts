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
  down: 'Service is down',
  log_error: 'Service is logging errors',
  agent_offline: 'Agent is offline',
  cpu_high: 'CPU over its threshold',
  mem_high: 'Memory over its threshold',
  disk_high: 'Disk over its threshold',
  temp_high: 'Temperature over its threshold',
  load_high: 'Load over its threshold',
  net_high: 'Network over its threshold',
  access_request: 'Somebody asks for credentials',
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
