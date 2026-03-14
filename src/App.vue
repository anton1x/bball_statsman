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
      <button :disabled="!canStart" @click="startSession()">Открыть матч</button>
      <p v-if="urlError" class="error">{{ urlError }}</p>

      <section v-if="savedVideos.length" class="saved-videos">
        <div class="toolbar">
          <h3>Ранее добавленные видео</h3>
        </div>
        <ul class="saved-videos-list">
          <li v-for="video in savedVideos" :key="video.url" class="saved-video-item">
            <div class="saved-video-meta">
              <button class="saved-video-link" @click="startSession(video.url)">{{ video.url }}</button>
              <p class="saved-video-details">
                Событий: {{ video.eventsCount }} · Обновлено: {{ formatDateTime(video.updatedAt) }}
              </p>
            </div>
            <button class="secondary" @click="removeSavedVideo(video.url)">Удалить</button>
          </li>
        </ul>
      </section>
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
        <div class="player-frame-wrap" v-if="embedUrl">
          <iframe
            ref="playerFrameRef"
            :src="embedUrl"
            width="100%"
            height="520"
            frameborder="0"
            allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
            allowfullscreen
            @load="onPlayerLoad"
          ></iframe>
          <transition name="event-pop">
            <div
              v-if="animatedEvent"
              :key="animatedEvent.id"
              class="player-event-float"
              :aria-label="`Событие: ${animatedEvent.label}`"
            >
              <span>{{ animatedEvent.icon }}</span>
            </div>
          </transition>
        </div>
        <p v-else class="error">
          Не удалось собрать embed-ссылку. Проверьте формат URL и попробуйте снова.
        </p>
      </div>

      <div class="card game-marker-card">
        <div class="toolbar">
          <h3>Игры внутри видео</h3>
          <button class="secondary" :disabled="!hasSyncedTime" @click="toggleGameBoundary">
            {{ activeGame ? 'Конец игры' : 'Начало игры' }}
          </button>
        </div>
        <p class="hint" v-if="activeGame">Сейчас идет игра #{{ activeGame.number }} (старт {{ formatSeconds(activeGame.startSec) }}).</p>
        <p class="hint" v-else>Нажмите «Начало игры», чтобы отметить следующий игровой отрезок.</p>

        <ul v-if="games.length" class="games-list">
          <li v-for="game in games" :key="game.id" class="games-list-item">
            <button class="time-link" @click="seekToGame(game)">{{ formatSeconds(game.startSec) }}</button>
            <span class="event-name">Игра #{{ game.number }}</span>
            <span class="event-game-label" v-if="game.endSec !== null">до {{ formatSeconds(game.endSec) }}</span>
            <span class="event-game-label" v-else>в процессе</span>
            <button class="secondary delete-event-button" @click="removeGame(game.number)">Удалить</button>
          </li>
        </ul>
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
          <div class="games-filter-inline" v-if="games.length">
            <button class="secondary" @click="selectPreviousGameFilter" :disabled="!canSelectPreviousGameFilter">←</button>
            <select v-model="selectedGameFilter" aria-label="Фильтр по играм">
              <option value="all">Все игры</option>
              <option v-for="game in games" :key="`filter-game-${game.number}`" :value="String(game.number)">Игра {{ game.number }}</option>
            </select>
            <button class="secondary" @click="selectNextGameFilter" :disabled="!canSelectNextGameFilter">→</button>
          </div>
          <button class="secondary" @click="clearEvents" :disabled="events.length === 0">Очистить</button>
        </div>
      </div>

      <ul v-if="logsViewMode === 'history' && filteredEvents.length" class="event-list">
        <li v-for="event in filteredEvents" :key="event.id" class="event-item">
          <button class="time-link" @click="seekToEvent(event.videoTimeSec)">{{ formatSeconds(event.videoTimeSec) }}</button>
          <span :class="['event-name', toneClass(event.type)]">{{ eventLabel(event.type) }}</span>
          <span v-if="eventGameLabel(event.videoTimeSec)" class="event-game-label">игра #{{ eventGameLabel(event.videoTimeSec) }}</span>
          <button class="secondary delete-event-button" @click="removeEvent(event.id)">Удалить</button>
        </li>
      </ul>
      <p v-else-if="logsViewMode === 'history'" class="hint">Пока нет событий для выбранных игр.</p>

      <div v-else class="stats-grid">
        <article class="stat-card">
          <p class="stat-label">Очки</p>
          <p class="stat-value">{{ summaryStats.points }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Ассисты</p>
          <p class="stat-value">{{ summaryStats.assists }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Подборы</p>
          <p class="stat-value">{{ summaryStats.rebounds }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Потери</p>
          <p class="stat-value">{{ summaryStats.turnovers }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Перехваты</p>
          <p class="stat-value">{{ summaryStats.steals }}</p>
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
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

const videoUrl = ref('');
const embedUrl = ref('');
const urlError = ref('');
const isSessionStarted = ref(false);
const playerFrameRef = ref(null);
const isSettingsOpen = ref(false);
const logsViewMode = ref('history');
const activeVideoUrl = ref('');
const gameRanges = ref([]);
const selectedGameFilter = ref('all');

const events = ref([]);
const currentTimeSec = ref(0);
const hasSyncedTime = ref(false);
const animatedEvent = ref(null);
const animatedEventRenderKey = ref(0);

let syncInterval = null;
let animationTimeout = null;
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
    id: 'possesion',
    label: 'Владение',
    events: [
      { type: 'assist', label: 'Ассист', tone: 'positive', icon: '🫶' },
      { type: 'turnover', label: 'Потеря', tone: 'negative', icon: '🚫' },
    ],
  },
  {
    id: 'defense',
    label: 'Защита',
    events: [
      {type: 'steal', label: 'Перехват', tone: 'positive', icon: '🥷'}
    ],
  },
  {
    id: 'rebounds',
    label: 'Подборы',
    events: [{ type: 'rebound', label: 'Подбор', tone: 'positive', icon: '👐' }],
  },
];

const eventTypes = eventGroups.flatMap((group) => group.events);
const storageKey = 'bball-statsman:v1';

function defaultGroupVisibility() {
  return Object.fromEntries(eventGroups.map((group) => [group.id, true]));
}

function defaultEventVisibility() {
  return Object.fromEntries(eventTypes.map((event) => [event.type, true]));
}

const groupVisibility = ref(defaultGroupVisibility());
const eventVisibility = ref(defaultEventVisibility());

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
const savedVideos = ref(loadSavedVideos());

const canStart = computed(() => Boolean(videoUrl.value));
const serializedEvents = computed(() => JSON.stringify(events.value, null, 2));
const games = computed(() =>
  gameRanges.value
    .filter((game) => typeof game?.startSec === 'number')
    .map((game, index) => ({
      id: game.id || `game-${index + 1}`,
      number: index + 1,
      startSec: Math.max(0, Math.floor(game.startSec)),
      endSec: typeof game.endSec === 'number' ? Math.max(0, Math.floor(game.endSec)) : null,
    })),
);

const activeGame = computed(() => games.value.find((game) => game.endSec === null) || null);

const filteredEvents = computed(() =>
  events.value.filter((event) => {
    const gameNumber = eventGameLabel(event.videoTimeSec);
    if (!gameNumber) {
      return true;
    }

    return selectedGameFilter.value === 'all' || selectedGameFilter.value === String(gameNumber);
  }),
);

const summaryStats = computed(() =>
  filteredEvents.value.reduce(
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

      if (event.type === 'steal') {
        acc.steals += 1;
      }

      if (event.type === 'assist') {
        acc.assists += 1;
      }

      return acc;
    },
    { points: 0, rebounds: 0, turnovers: 0, steals: 0, assists: 0 },
  ),
);

function setEventVisibility(type, isVisible) {
  eventVisibility.value[type] = isVisible;
}

function normalizeVideoUrl(url) {
  const safeUrl = typeof url === 'string' ? url : '';

  try {
    return new URL(safeUrl).toString();
  } catch {
    return safeUrl.trim();
  }
}

function loadSavedVideos() {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) {
      return [];
    }

    const parsed = JSON.parse(raw);
    if (!parsed?.videos || typeof parsed.videos !== 'object') {
      return [];
    }

    return Object.values(parsed.videos)
      .filter((video) => typeof video?.url === 'string')
      .map((video) => ({
        url: video.url,
        eventsCount: Array.isArray(video.events) ? video.events.length : 0,
        updatedAt: typeof video.updatedAt === 'number' ? video.updatedAt : 0,
      }))
      .sort((a, b) => b.updatedAt - a.updatedAt);
  } catch {
    return [];
  }
}

function loadVideoState(videoUrlToLoad) {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw);
    return parsed?.videos?.[videoUrlToLoad] || null;
  } catch {
    return null;
  }
}

function persistVideoState() {
  if (!activeVideoUrl.value) {
    return;
  }

  try {
    const raw = localStorage.getItem(storageKey);
    const parsed = raw ? JSON.parse(raw) : {};
    const videos = parsed?.videos && typeof parsed.videos === 'object' ? parsed.videos : {};

    videos[activeVideoUrl.value] = {
      url: activeVideoUrl.value,
      updatedAt: Date.now(),
      events: events.value,
      games: gameRanges.value,
      settings: {
        groupVisibility: groupVisibility.value,
        eventVisibility: eventVisibility.value,
        selectedGameFilter: selectedGameFilter.value,
      },
    };

    localStorage.setItem(storageKey, JSON.stringify({ videos }));
    savedVideos.value = loadSavedVideos();
  } catch {
    // ignore storage errors
  }
}

function removeSavedVideo(videoUrlToRemove) {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) {
      return;
    }

    const parsed = JSON.parse(raw);
    if (!parsed?.videos || typeof parsed.videos !== 'object') {
      return;
    }

    delete parsed.videos[videoUrlToRemove];
    localStorage.setItem(storageKey, JSON.stringify({ videos: parsed.videos }));
    savedVideos.value = loadSavedVideos();
  } catch {
    // ignore storage errors
  }
}

function applyStoredVideoState(videoState) {
  events.value = Array.isArray(videoState?.events) ? videoState.events : [];
  gameRanges.value = Array.isArray(videoState?.games) ? videoState.games : [];

  groupVisibility.value = {
    ...defaultGroupVisibility(),
    ...(videoState?.settings?.groupVisibility || {}),
  };
  eventVisibility.value = {
    ...defaultEventVisibility(),
    ...(videoState?.settings?.eventVisibility || {}),
  };

  const availableGameIds = games.value.map((game) => String(game.number));
  const storedFilter = String(videoState?.settings?.selectedGameFilter || 'all');
  selectedGameFilter.value = storedFilter === 'all' || availableGameIds.includes(storedFilter) ? storedFilter : 'all';
}

function resetVideoState() {
  events.value = [];
  gameRanges.value = [];
  selectedGameFilter.value = 'all';
  groupVisibility.value = defaultGroupVisibility();
  eventVisibility.value = defaultEventVisibility();
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

function startSession(selectedUrl = videoUrl.value) {
  const normalizedUrl = normalizeVideoUrl(selectedUrl);
  const parsedUrl = parseVkEmbedUrl(normalizedUrl);

  if (!parsedUrl) {
    urlError.value = 'Не удалось распознать ссылку. Нужен URL с фрагментом video-oid_id.';
    return;
  }

  urlError.value = '';
  activeVideoUrl.value = normalizedUrl;
  videoUrl.value = normalizedUrl;
  embedUrl.value = parsedUrl;
  currentTimeSec.value = 0;
  hasSyncedTime.value = false;
  animatedEvent.value = null;
  applyStoredVideoState(loadVideoState(normalizedUrl));
  if (selectedGameFilter.value !== 'all' && !games.value.some((game) => String(game.number) === selectedGameFilter.value)) {
    selectedGameFilter.value = 'all';
  }
  isSessionStarted.value = true;
  persistVideoState();
}

function resetSession() {
  stopSync();
  vkPlayer = null;
  isSessionStarted.value = false;
  activeVideoUrl.value = '';
  embedUrl.value = '';
  currentTimeSec.value = 0;
  hasSyncedTime.value = false;
  animatedEvent.value = null;
  resetVideoState();
}

function postPlayerCommand(payload) {
  const target = playerFrameRef.value?.contentWindow;
  if (!target) {
    return;
  }

  target.postMessage(JSON.stringify(payload), '*');
}

async function playVideo() {
  if (vkPlayer?.play) {
    try {
      await vkPlayer.play();
      return;
    } catch {
      // fallback to postMessage API
    }
  }

  postPlayerCommand({ type: 'play' });
  postPlayerCommand({ type: 'vk_player_play' });
  postPlayerCommand({ method: 'play' });
  postPlayerCommand({ event: 'command', func: 'play' });
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

async function seekTo(timeSec, shouldPlay = false) {
  const safeTime = Math.max(0, Math.floor(timeSec));

  if (vkPlayer?.seek) {
    try {
      await vkPlayer.seek(safeTime);
      currentTimeSec.value = safeTime;
      hasSyncedTime.value = true;

      if (shouldPlay) {
        await playVideo();
      }

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

  if (shouldPlay) {
    await playVideo();
  }
}

function seekToEvent(eventTimeSec) {
  const rewoundTime = Math.max(0, Math.floor(eventTimeSec) - 2);
  seekTo(rewoundTime, true);
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

function removeEvent(eventId) {
  events.value = events.value.filter((event) => event.id !== eventId);
}

function toggleGameBoundary() {
  if (!hasSyncedTime.value) {
    return;
  }

  const openGame = activeGame.value;
  if (openGame) {
    gameRanges.value = gameRanges.value.map((game, index) => {
      if (index !== openGame.number - 1) {
        return game;
      }

      return {
        ...game,
        endSec: Math.max(openGame.startSec, currentTimeSec.value),
      };
    });
    return;
  }

  gameRanges.value.push({
    id: crypto.randomUUID(),
    startSec: currentTimeSec.value,
    endSec: null,
  });

}


function eventGameLabel(timeSec) {
  const second = Math.max(0, Math.floor(timeSec));
  const game = games.value.find((candidate) => {
    const isAfterStart = second >= candidate.startSec;
    const isBeforeEnd = candidate.endSec === null ? true : second <= candidate.endSec;
    return isAfterStart && isBeforeEnd;
  });

  return game?.number || null;
}

function seekToGame(game) {
  const rewoundTime = Math.max(0, game.startSec - 2);
  seekTo(rewoundTime, true);
}

function removeGame(gameNumber) {
  gameRanges.value = gameRanges.value.filter((_, index) => index !== gameNumber - 1);
}

const gameFilterOptions = computed(() => ['all', ...games.value.map((game) => String(game.number))]);
const selectedGameFilterIndex = computed(() => gameFilterOptions.value.indexOf(selectedGameFilter.value));
const canSelectPreviousGameFilter = computed(() => selectedGameFilterIndex.value > 0);
const canSelectNextGameFilter = computed(() => selectedGameFilterIndex.value >= 0 && selectedGameFilterIndex.value < gameFilterOptions.value.length - 1);

function selectPreviousGameFilter() {
  if (!canSelectPreviousGameFilter.value) {
    return;
  }

  selectedGameFilter.value = gameFilterOptions.value[selectedGameFilterIndex.value - 1];
}

function selectNextGameFilter() {
  if (!canSelectNextGameFilter.value) {
    return;
  }

  selectedGameFilter.value = gameFilterOptions.value[selectedGameFilterIndex.value + 1];
}

function clearEvents() {
  const isConfirmed = window.confirm('Вы точно хотите удалить все события?');

  if (!isConfirmed) {
    return;
  }

  events.value = [];
}

function triggerEventAnimationForSecond(second) {
  const lastEventAtSecond = [...events.value].reverse().find((event) => event.videoTimeSec === second);

  if (!lastEventAtSecond) {
    return;
  }

  const meta = eventMetaByType[lastEventAtSecond.type];
  animatedEvent.value = {
    id: `${lastEventAtSecond.id}-${animatedEventRenderKey.value}`,
    icon: meta?.icon || '🏀',
    label: meta?.label || lastEventAtSecond.type,
  };
  animatedEventRenderKey.value += 1;

  if (animationTimeout) {
    clearTimeout(animationTimeout);
  }

  animationTimeout = setTimeout(() => {
    animatedEvent.value = null;
  }, 1000);
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

function formatDateTime(timestamp) {
  if (!timestamp) {
    return '—';
  }

  return new Intl.DateTimeFormat('ru-RU', {
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(timestamp);
}

watch(currentTimeSec, (nextSecond, previousSecond) => {
  if (!hasSyncedTime.value || nextSecond === previousSecond) {
    return;
  }

  triggerEventAnimationForSecond(nextSecond);
});

watch(
  [events, gameRanges, selectedGameFilter, groupVisibility, eventVisibility],
  () => {
    if (!isSessionStarted.value) {
      return;
    }

    persistVideoState();
  },
  { deep: true },
);

watch(
  games,
  (nextGames) => {
    const available = nextGames.map((game) => String(game.number));

    if (selectedGameFilter.value !== 'all' && !available.includes(selectedGameFilter.value)) {
      selectedGameFilter.value = 'all';
    }
  },
  { deep: true },
);

onMounted(() => {
  window.addEventListener('message', handlePlayerMessage);
});

onBeforeUnmount(() => {
  stopSync();
  vkPlayer = null;
  if (animationTimeout) {
    clearTimeout(animationTimeout);
    animationTimeout = null;
  }
  animatedEvent.value = null;
  window.removeEventListener('message', handlePlayerMessage);
});
</script>
