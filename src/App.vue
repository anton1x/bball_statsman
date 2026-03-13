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
          height="390"
          frameborder="0"
          allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
          allowfullscreen
          @load="onPlayerLoad"
        ></iframe>
        <p v-else class="error">
          Не удалось собрать embed-ссылку. Проверьте формат URL и попробуйте снова.
        </p>
      </div>

      <div class="card controls-card">
        <h2>Текущее время в видео</h2>
        <div class="timer-row">
          <div class="timer">{{ formatSeconds(currentTimeSec) }}</div>
          <button class="secondary" @click="requestCurrentTime">Обновить время</button>
        </div>
        <p class="hint">{{ syncHint }}</p>

        <h2>События</h2>
        <div class="events-grid">
          <button
            v-for="event in eventTypes"
            :key="event.type"
            :disabled="!hasSyncedTime"
            @click="addEvent(event.type)"
          >
            {{ event.label }}
          </button>
        </div>
      </div>
    </section>

    <section v-if="isSessionStarted" class="card logs-card">
      <div class="toolbar">
        <h2>Список зафиксированных событий</h2>
        <button class="secondary" @click="clearEvents" :disabled="events.length === 0">Очистить</button>
      </div>

      <ul v-if="events.length" class="event-list">
        <li v-for="event in events" :key="event.id" class="event-item">
          <button class="time-link" @click="seekTo(event.videoTimeSec)">{{ formatSeconds(event.videoTimeSec) }}</button>
          <span :class="['event-name', toneClass(event.type)]">{{ eventLabel(event.type) }}</span>
        </li>
      </ul>
      <p v-else class="hint">Пока нет событий — нажмите одну из кнопок выше.</p>

      <div v-if="isDebugMode" class="debug-block">
        <h3>Debug: JSON для отправки на backend</h3>
        <pre>{{ serializedEvents }}</pre>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

const videoUrl = ref('');
const embedUrl = ref('');
const urlError = ref('');
const isSessionStarted = ref(false);
const playerFrameRef = ref(null);

const events = ref([]);
const currentTimeSec = ref(0);
const hasSyncedTime = ref(false);

let syncInterval = null;

const eventTypes = [
  { type: 'made_2pt', label: '2-очковое попадание', tone: 'positive' },
  { type: 'made_3pt', label: '3-очковое попадание', tone: 'positive' },
  { type: 'missed_shot', label: 'Неудачный бросок', tone: 'negative' },
  { type: 'rebound', label: 'Подбор', tone: 'positive' },
  { type: 'turnover', label: 'Потеря', tone: 'negative' },
];

const eventMetaByType = Object.fromEntries(eventTypes.map((event) => [event.type, event]));
const isDebugMode = new URLSearchParams(window.location.search).get('debug') === '1';

const canStart = computed(() => Boolean(videoUrl.value));
const serializedEvents = computed(() => JSON.stringify(events.value, null, 2));
const syncHint = computed(() =>
  hasSyncedTime.value
    ? 'Время берется из VK плеера. При клике на событие фиксируется текущий таймкод видео.'
    : 'Ожидаем синхронизацию с плеером. Запустите видео и нажмите «Обновить время».',
);

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

function requestCurrentTime() {
  postPlayerCommand({ type: 'vk_player_get_current_time' });
  postPlayerCommand({ type: 'getCurrentTime' });
  postPlayerCommand({ method: 'getCurrentTime' });
}

function seekTo(timeSec) {
  const safeTime = Math.max(0, Math.floor(timeSec));

  // VK embeds can expose slightly different message contracts; send several compatible shapes.
  postPlayerCommand({ type: 'setCurrentTime', data: safeTime });
  postPlayerCommand({ type: 'setCurrentTime', time: safeTime });
  postPlayerCommand({ type: 'vk_player_set_current_time', time: safeTime });
  postPlayerCommand({ method: 'setCurrentTime', value: safeTime });
  postPlayerCommand({ method: 'setCurrentTime', args: [safeTime] });
  postPlayerCommand({ event: 'command', func: 'setCurrentTime', args: [safeTime] });

  currentTimeSec.value = safeTime;
  requestCurrentTime();
}

function onPlayerLoad() {
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
  window.removeEventListener('message', handlePlayerMessage);
});
</script>
