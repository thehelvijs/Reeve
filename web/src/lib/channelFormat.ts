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
