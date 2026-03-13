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
      <p class="hint">
        Поддерживаются ссылки вида <code>.../video-oid_id</code>. Для тайминга используйте встроенный секундомер,
        при необходимости можно вручную поправить время события.
      </p>
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
          :src="embedUrl"
          width="100%"
          height="390"
          frameborder="0"
          allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
          allowfullscreen
        ></iframe>
        <p v-else class="error">
          Не удалось собрать embed-ссылку. Проверьте формат URL и попробуйте снова.
        </p>
      </div>

      <div class="card controls-card">
        <h2>Текущее время</h2>
        <div class="timer-row">
          <div class="timer">{{ formatSeconds(currentTimeSec) }}</div>
          <div class="timer-buttons">
            <button class="secondary" @click="toggleTimer">{{ timerRunning ? 'Пауза' : 'Старт' }}</button>
            <button class="secondary" @click="adjustTime(-5)">-5 сек</button>
            <button class="secondary" @click="adjustTime(5)">+5 сек</button>
          </div>
        </div>

        <label for="manualTime">Ручная корректировка (сек)</label>
        <input id="manualTime" type="number" min="0" :value="currentTimeSec" @change="setManualTime" />

        <h2>События</h2>
        <div class="events-grid">
          <button v-for="event in eventTypes" :key="event.type" @click="addEvent(event.type)">
            {{ event.label }}
          </button>
        </div>
      </div>
    </section>

    <section v-if="isSessionStarted" class="card logs-card">
      <div class="toolbar">
        <h2>Лента событий</h2>
        <button class="secondary" @click="clearEvents" :disabled="events.length === 0">Очистить</button>
      </div>

      <table v-if="events.length">
        <thead>
          <tr>
            <th>#</th>
            <th>Время</th>
            <th>Тип</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(event, index) in events" :key="event.id">
            <td>{{ index + 1 }}</td>
            <td>{{ formatSeconds(event.videoTimeSec) }}</td>
            <td>{{ event.type }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="hint">Пока нет событий — нажмите одну из кнопок выше.</p>

      <h3>JSON для отправки на backend</h3>
      <pre>{{ serializedEvents }}</pre>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue';

const videoUrl = ref('');
const embedUrl = ref('');
const urlError = ref('');
const isSessionStarted = ref(false);

const events = ref([]);
const currentTimeSec = ref(0);
const timerRunning = ref(false);
let timerInterval = null;

const eventTypes = [
  { type: 'made_shot', label: 'Успешный бросок' },
  { type: 'missed_shot', label: 'Неудачный бросок' },
  { type: 'rebound', label: 'Подбор' },
  { type: 'turnover', label: 'Потеря' },
];

const canStart = computed(() => Boolean(videoUrl.value));
const serializedEvents = computed(() => JSON.stringify(events.value, null, 2));

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
  isSessionStarted.value = true;
}

function resetSession() {
  stopTimer();
  isSessionStarted.value = false;
  embedUrl.value = '';
  currentTimeSec.value = 0;
}

function addEvent(type) {
  events.value.push({
    id: crypto.randomUUID(),
    videoTimeSec: currentTimeSec.value,
    type,
  });
}

function clearEvents() {
  events.value = [];
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

function toggleTimer() {
  if (timerRunning.value) {
    stopTimer();
    return;
  }

  timerRunning.value = true;
  timerInterval = setInterval(() => {
    currentTimeSec.value += 1;
  }, 1000);
}

function stopTimer() {
  timerRunning.value = false;
  if (timerInterval) {
    clearInterval(timerInterval);
    timerInterval = null;
  }
}

function adjustTime(delta) {
  currentTimeSec.value = Math.max(0, currentTimeSec.value + delta);
}

function setManualTime(event) {
  const value = Number(event.target.value);
  currentTimeSec.value = Number.isFinite(value) ? Math.max(0, Math.floor(value)) : 0;
}

onBeforeUnmount(() => {
  stopTimer();
});
</script>
