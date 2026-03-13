<template>
  <div class="app">
    <header>
      <h1>Basketball Statsman</h1>
      <p>Фиксируйте игровые события по ходу просмотра записи матча.</p>
    </header>

    <section v-if="!isSessionStarted" class="card start-card">
      <label for="videoUrl">Ссылка на VK Video</label>
      <input
        id="videoUrl"
        v-model.trim="videoUrl"
        type="url"
        placeholder="https://vkvideo.ru/video-123_456"
      />
      <p class="hint">Поддерживаются ссылки вида <code>.../video-oid_id</code>.</p>
      <button :disabled="!canStart" @click="startSession">Открыть матч</button>
      <p v-if="urlError" class="error">{{ urlError }}</p>
    </section>

    <section v-else class="main-grid">
      <div class="card controls-card">
        <div class="toolbar">
          <h2>События</h2>
          <button class="secondary" @click="isSettingsOpen = true">Настройки</button>
        </div>

        <div class="event-blocks">
          <section v-for="group in visibleEventGroups" :key="group.id" class="event-group">
            <h3>{{ group.label }}</h3>
            <div class="events-grid">
              <button
                v-for="event in group.events"
                :key="event.type"
                :class="['event-button', `event-${event.tone}`]"
                :disabled="!hasSyncedTime"
                @click="addEvent(event.type)"
              >
                <span class="event-icon" aria-hidden="true">{{ event.icon }}</span>
                <span class="event-label">{{ event.label }}</span>
              </button>
            </div>
          </section>
        </div>
      </div>

      <div class="card player-card">
        <div class="toolbar">
          <strong>Видео</strong>
          <button class="secondary" @click="resetSession">Сменить ссылку</button>
        </div>
        <iframe
          v-if="embedUrl"
          ref="playerFrameRef"
          :src="embedUrl"
          width="100%"
          height="520"
          frameborder="0"
          allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
          allowfullscreen
          @load="onPlayerLoad"
        ></iframe>
        <p v-else class="error">
          Не удалось собрать embed-ссылку. Проверьте формат URL и попробуйте снова.
        </p>
      </div>
    </section>

    <section v-if="isSessionStarted" class="card logs-card">
      <div class="toolbar">
        <h2>{{ logsViewMode === 'history' ? 'История событий' : 'Статистика' }}</h2>
        <div class="toolbar-actions">
          <div class="view-switch" role="tablist" aria-label="Переключение вида блока событий">
            <button
              :class="['secondary', { active: logsViewMode === 'history' }]"
              role="tab"
              :aria-selected="logsViewMode === 'history'"
              @click="logsViewMode = 'history'"
            >
              История
            </button>
            <button
              :class="['secondary', { active: logsViewMode === 'stats' }]"
              role="tab"
              :aria-selected="logsViewMode === 'stats'"
              @click="logsViewMode = 'stats'"
            >
              Статистика
            </button>
          </div>
          <button class="secondary" @click="clearEvents" :disabled="events.length === 0">Очистить</button>
        </div>
      </div>

      <ul v-if="logsViewMode === 'history' && events.length" class="event-list">
        <li v-for="event in events" :key="event.id" class="event-item">
          <button class="time-link" @click="seekTo(event.videoTimeSec)">{{ formatSeconds(event.videoTimeSec) }}</button>
          <span :class="['event-name', toneClass(event.type)]">{{ eventLabel(event.type) }}</span>
        </li>
      </ul>
      <p v-else-if="logsViewMode === 'history'" class="hint">Пока нет событий — нажмите одну из кнопок выше.</p>

      <div v-else class="stats-grid">
        <article class="stat-card">
          <p class="stat-label">Очки</p>
          <p class="stat-value">{{ summaryStats.points }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Подборы</p>
          <p class="stat-value">{{ summaryStats.rebounds }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Потери</p>
          <p class="stat-value">{{ summaryStats.turnovers }}</p>
        </article>
      </div>

      <div v-if="isDebugMode" class="debug-block">
        <h3>Debug: JSON для отправки на backend</h3>
        <pre>{{ serializedEvents }}</pre>
      </div>
    </section>

    <div v-if="isSettingsOpen" class="settings-overlay" @click.self="isSettingsOpen = false">
      <section class="settings-modal card" role="dialog" aria-modal="true" aria-label="Настройки">
        <div class="toolbar">
          <h2>Настройки</h2>
          <button class="secondary" @click="isSettingsOpen = false">Закрыть</button>
        </div>

        <div class="settings-block">
          <h3>События</h3>
          <div class="settings-groups">
            <section v-for="group in eventGroups" :key="`settings-${group.id}`" class="settings-group">
              <label class="toggle-row group-toggle">
                <input v-model="groupVisibility[group.id]" type="checkbox" />
                <span>{{ group.label }}</span>
              </label>

              <div class="settings-events">
                <label v-for="event in group.events" :key="`settings-${event.type}`" class="toggle-row event-toggle">
                  <input
                    :checked="eventVisibility[event.type]"
                    type="checkbox"
                    :disabled="!groupVisibility[group.id]"
                    @change="setEventVisibility(event.type, $event.target.checked)"
                  />
                  <span>{{ event.icon }} {{ event.label }}</span>
                </label>
              </div>
            </section>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

const videoUrl = ref('');
const embedUrl = ref('');
const urlError = ref('');
const isSessionStarted = ref(false);
const playerFrameRef = ref(null);
const isSettingsOpen = ref(false);
const logsViewMode = ref('history');

const events = ref([]);
const currentTimeSec = ref(0);
const hasSyncedTime = ref(false);

let syncInterval = null;
let vkPlayer = null;

const eventGroups = [
  {
    id: 'shots',
    label: 'Броски',
    events: [
      { type: 'made_2pt', label: '2-очковое', tone: 'positive', icon: '🏀' },
      { type: 'made_3pt', label: '3-очковое', tone: 'positive', icon: '🎯' },
      { type: 'missed_shot', label: 'Промах', tone: 'negative', icon: '❌' },
    ],
  },
  {
    id: 'defense',
    label: 'Защита',
    events: [{ type: 'turnover', label: 'Потеря', tone: 'negative', icon: '🚫' }],
  },
  {
    id: 'rebounds',
    label: 'Подборы',
    events: [{ type: 'rebound', label: 'Подбор', tone: 'positive', icon: '👐' }],
  },
];

const eventTypes = eventGroups.flatMap((group) => group.events);
const groupVisibility = ref(Object.fromEntries(eventGroups.map((group) => [group.id, true])));
const eventVisibility = ref(Object.fromEntries(eventTypes.map((event) => [event.type, true])));

const visibleEventGroups = computed(() =>
  eventGroups
    .filter((group) => groupVisibility.value[group.id])
    .map((group) => ({
      ...group,
      events: group.events.filter((event) => eventVisibility.value[event.type]),
    }))
    .filter((group) => group.events.length > 0),
);

const eventMetaByType = Object.fromEntries(eventTypes.map((event) => [event.type, event]));
const isDebugMode = new URLSearchParams(window.location.search).get('debug') === '1';

const canStart = computed(() => Boolean(videoUrl.value));
const serializedEvents = computed(() => JSON.stringify(events.value, null, 2));
const summaryStats = computed(() =>
  events.value.reduce(
    (acc, event) => {
      if (event.type === 'made_2pt') {
        acc.points += 2;
      }

      if (event.type === 'made_3pt') {
        acc.points += 3;
      }

      if (event.type === 'rebound') {
        acc.rebounds += 1;
      }

      if (event.type === 'turnover') {
        acc.turnovers += 1;
      }

      return acc;
    },
    { points: 0, rebounds: 0, turnovers: 0 },
  ),
);

function setEventVisibility(type, isVisible) {
  eventVisibility.value[type] = isVisible;
}

function parseVkEmbedUrl(url) {
  try {
    const parsed = new URL(url);
    const match = parsed.pathname.match(/video(-?\d+)_(\d+)/);

    if (!match) {
      return '';
    }

    const oid = match[1];
    const id = match[2];
    const hash = parsed.searchParams.get('list') || '';

    const embed = new URL('https://vkvideo.ru/video_ext.php');
    embed.searchParams.set('oid', oid);
    embed.searchParams.set('id', id);
    embed.searchParams.set('js_api', '1');

    if (hash) {
      embed.searchParams.set('hash', hash);
    }

    return embed.toString();
  } catch {
    return '';
  }
}

function startSession() {
  const parsedUrl = parseVkEmbedUrl(videoUrl.value);

  if (!parsedUrl) {
    urlError.value = 'Не удалось распознать ссылку. Нужен URL с фрагментом video-oid_id.';
    return;
  }

  urlError.value = '';
  embedUrl.value = parsedUrl;
  currentTimeSec.value = 0;
  hasSyncedTime.value = false;
  isSessionStarted.value = true;
}

function resetSession() {
  stopSync();
  vkPlayer = null;
  isSessionStarted.value = false;
  embedUrl.value = '';
  currentTimeSec.value = 0;
  hasSyncedTime.value = false;
}

function postPlayerCommand(payload) {
  const target = playerFrameRef.value?.contentWindow;
  if (!target) {
    return;
  }

  target.postMessage(JSON.stringify(payload), '*');
}

async function requestCurrentTime() {
  if (vkPlayer?.getCurrentTime) {
    try {
      const time = await vkPlayer.getCurrentTime();
      if (typeof time === 'number' && Number.isFinite(time)) {
        currentTimeSec.value = Math.max(0, Math.floor(time));
        hasSyncedTime.value = true;
        return;
      }
    } catch {
      // fallback to postMessage API
    }
  }

  postPlayerCommand({ type: 'vk_player_get_current_time' });
  postPlayerCommand({ type: 'getCurrentTime' });
  postPlayerCommand({ method: 'getCurrentTime' });
}

async function seekTo(timeSec) {
  const safeTime = Math.max(0, Math.floor(timeSec));

  if (vkPlayer?.seek) {
    try {
      await vkPlayer.seek(safeTime);
      currentTimeSec.value = safeTime;
      hasSyncedTime.value = true;
      return;
    } catch {
      // fallback to postMessage API
    }
  }

  // Fallback for embeds where direct JS API object is not available.
  postPlayerCommand({ type: 'setCurrentTime', data: safeTime });
  postPlayerCommand({ type: 'setCurrentTime', time: safeTime });
  postPlayerCommand({ type: 'vk_player_set_current_time', time: safeTime });
  postPlayerCommand({ method: 'setCurrentTime', value: safeTime });
  postPlayerCommand({ method: 'setCurrentTime', args: [safeTime] });
  postPlayerCommand({ event: 'command', func: 'setCurrentTime', args: [safeTime] });

  currentTimeSec.value = safeTime;
  requestCurrentTime();
}

async function onPlayerLoad() {
  const iframe = playerFrameRef.value;

  if (iframe && window.VK?.VideoPlayer) {
    try {
      vkPlayer = await window.VK.VideoPlayer(iframe);
    } catch {
      vkPlayer = null;
    }
  }

  startSync();
  requestCurrentTime();
}

function startSync() {
  stopSync();
  syncInterval = setInterval(() => {
    requestCurrentTime();
  }, 1000);
}

function stopSync() {
  if (syncInterval) {
    clearInterval(syncInterval);
    syncInterval = null;
  }
}

function handlePlayerMessage(event) {
  if (!event.origin.includes('vkvideo.ru') && !event.origin.includes('vk.com')) {
    return;
  }

  let payload = event.data;

  if (typeof payload === 'string') {
    try {
      payload = JSON.parse(payload);
    } catch {
      return;
    }
  }

  const candidateTime =
    payload?.current_time ?? payload?.currentTime ?? payload?.time ?? payload?.data?.current_time ?? payload?.data?.time;

  if (typeof candidateTime === 'number' && Number.isFinite(candidateTime)) {
    currentTimeSec.value = Math.max(0, Math.floor(candidateTime));
    hasSyncedTime.value = true;
  }
}

function addEvent(type) {
  if (!hasSyncedTime.value) {
    return;
  }

  events.value.push({
    id: crypto.randomUUID(),
    videoTimeSec: currentTimeSec.value,
    type,
  });
}

function clearEvents() {
  events.value = [];
}

function eventLabel(type) {
  return eventMetaByType[type]?.label || type;
}

function toneClass(type) {
  return `tone-${eventMetaByType[type]?.tone || 'neutral'}`;
}

function formatSeconds(total) {
  const hours = Math.floor(total / 3600)
    .toString()
    .padStart(2, '0');
  const minutes = Math.floor((total % 3600) / 60)
    .toString()
    .padStart(2, '0');
  const seconds = Math.floor(total % 60)
    .toString()
    .padStart(2, '0');

  return `${hours}:${minutes}:${seconds}`;
}

onMounted(() => {
  window.addEventListener('message', handlePlayerMessage);
});

onBeforeUnmount(() => {
  stopSync();
  vkPlayer = null;
  window.removeEventListener('message', handlePlayerMessage);
});
</script>
