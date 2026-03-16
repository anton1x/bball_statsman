<template>
  <div class="app">
    <header>
      <h1>Basketball Statsman</h1>
      <p>Фиксируйте игровые события по ходу просмотра записи матча.</p>
    </header>

    <transition name="copy-toast">
      <div v-if="shareLinkStatus" class="copy-toast" role="status" aria-live="polite">{{ shareLinkStatus }}</div>
    </transition>

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
          <div class="toolbar-actions">
            <button class="secondary" @click="isTeamsOpen = true">Команды</button>
            <button class="secondary" @click="isSettingsOpen = true">Настройки</button>
          </div>
        </div>

        <div class="player-selector" v-if="playerOptions.length">
          <button class="secondary" @click="selectPreviousPlayer" :disabled="!canSelectPreviousPlayer">←</button>
          <select v-model="selectedPlayerId" aria-label="Выбор игрока для события">
            <option v-for="option in playerOptions" :key="option.id" :value="option.id">
              {{ option.teamName }} · {{ option.playerName }} ({{ option.shortcut }})
            </option>
          </select>
          <button class="secondary" @click="selectNextPlayer" :disabled="!canSelectNextPlayer">→</button>
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
                <span class="event-label">{{ event.label }} <small class="event-shortcut">{{ eventShortcutLabel(event.type) }}</small></span>
              </button>
            </div>
          </section>
        </div>
      </div>

      <div class="card player-card">
        <div class="toolbar">
          <strong>Видео</strong>
          <div class="toolbar-actions">
            <button class="secondary" @click="copyShareLink" :disabled="!activeVideoUrl">Копировать ссылку</button>
            <button class="secondary" @click="resetSession">Сменить ссылку</button>
          </div>
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

    </section>

    <section v-if="isSessionStarted" class="logs-layout">
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

      <section class="card logs-card">
      <h2 class="logs-title">{{ logsViewMode === 'history' ? 'История событий' : 'Статистика' }}</h2>
      <div class="logs-toolbar">
        <div class="toolbar-actions logs-toolbar-row">
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
          <button v-if="logsViewMode === 'history'" class="secondary" @click="clearEvents" :disabled="events.length === 0">Очистить</button>
        </div>
        <div class="toolbar-actions logs-toolbar-row logs-filters-row">
          <div class="games-filter-inline" v-if="games.length">
            <button class="secondary" @click="selectPreviousGameFilter" :disabled="!canSelectPreviousGameFilter">←</button>
            <select v-model="selectedGameFilter" aria-label="Фильтр по играм">
              <option value="all">Все игры</option>
              <option v-for="game in games" :key="`filter-game-${game.number}`" :value="String(game.number)">Игра {{ game.number }}</option>
            </select>
            <button class="secondary" @click="selectNextGameFilter" :disabled="!canSelectNextGameFilter">→</button>
          </div>
          <div class="games-filter-inline" v-if="rosterFilterOptions.length">
            <button class="secondary" @click="selectPreviousRosterFilter" :disabled="!canSelectPreviousRosterFilter">←</button>
            <select v-model="selectedRosterFilter" aria-label="Фильтр по командам и игрокам">
              <option v-for="option in rosterFilterOptions" :key="`roster-filter-${option.value}`" :value="option.value">
                {{ option.label }}
              </option>
            </select>
            <button class="secondary" @click="selectNextRosterFilter" :disabled="!canSelectNextRosterFilter">→</button>
          </div>
          <button
            v-if="logsViewMode === 'history'"
            :class="['icon-toggle', 'highlight-filter-toggle', { active: showOnlyHighlights }]"
            @click="showOnlyHighlights = !showOnlyHighlights"
            :title="showOnlyHighlights ? 'Показаны только выдающиеся' : 'Показывать только выдающиеся'"
            aria-label="Фильтр выдающихся событий"
          >
            🔥
          </button>
        </div>
      </div>

      <ul v-if="logsViewMode === 'history' && filteredEvents.length" class="event-list">
        <li v-for="event in filteredEvents" :key="event.id" class="event-item">
          <button class="time-link" @click="seekToEvent(event.videoTimeSec)">{{ formatSeconds(event.videoTimeSec) }}</button>
          <span :class="['event-name', toneClass(event.type)]">{{ eventLabel(event.type) }}</span>
          <span v-if="!isUnknownEventPlayer(event)" class="event-player-label">
            <span class="team-color-dot" :style="{ backgroundColor: eventPlayerTeamColor(event) }" aria-hidden="true"></span>
            {{ eventPlayerName(event) }}
          </span>
          <div v-else class="unknown-player-assign">
            <button
              v-if="assigningPlayerEventId !== event.id"
              class="event-player-link"
              @click="startAssignEventPlayer(event.id)"
            >
              {{ eventPlayerLabel(event) }}
            </button>
            <select
              v-else
              class="event-player-select"
              :value="event.playerId || ''"
              @change="assignEventPlayer(event.id, $event.target.value)"
              @blur="cancelAssignEventPlayer"
            >
              <option disabled value="">Выберите игрока</option>
              <option v-for="option in playerOptions" :key="`event-player-${event.id}-${option.id}`" :value="option.id">
                {{ option.teamName }} · {{ option.playerName }}
              </option>
            </select>
          </div>
          <span v-if="eventGameLabel(event.videoTimeSec)" class="event-game-label">игра #{{ eventGameLabel(event.videoTimeSec) }}</span>
          <button
            :class="['icon-toggle', 'event-highlight-button', { active: event.isHighlighted }]"
            :aria-pressed="event.isHighlighted"
            @click="toggleEventHighlight(event.id)"
            :title="event.isHighlighted ? 'Убрать из выдающихся' : 'Отметить как выдающееся'"
            aria-label="Переключить выдающееся событие"
          >
            🔥
          </button>
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
          <p class="stat-label">Блокшоты</p>
          <p class="stat-value">{{ summaryStats.blocks }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Потери</p>
          <p class="stat-value">{{ summaryStats.turnovers }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">Перехваты</p>
          <p class="stat-value">{{ summaryStats.steals }}</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">3-очковые</p>
          <p class="stat-value">{{ summaryStats.made_3pt }}/{{ summaryStats.attempt_3pt }} ({{ summaryStats.percentage_3pt.toFixed(2) }}%)</p>
        </article>
        <article class="stat-card">
          <p class="stat-label">2-очковые</p>
          <p class="stat-value">{{ summaryStats.made_2pt }}/{{ summaryStats.attempt_2pt }} ({{ summaryStats.percentage_2pt.toFixed(2) }}%)</p>
        </article>
      </div>

      <div v-if="isDebugMode" class="debug-block">
        <h3>Debug: JSON для отправки на backend</h3>
        <pre>{{ serializedEvents }}</pre>
      </div>
    </section>
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

    <div v-if="isTeamsOpen" class="settings-overlay" @click.self="isTeamsOpen = false">
      <section class="settings-modal card" role="dialog" aria-modal="true" aria-label="Команды и игроки">
        <div class="toolbar">
          <h2>Команды и игроки</h2>
          <button class="secondary" @click="isTeamsOpen = false">Закрыть</button>
        </div>

        <div class="teams-toolbar">
          <button @click="addTeam">Добавить команду</button>
        </div>

        <div class="teams-list">
          <section v-for="team in teams" :key="team.id" class="settings-group">
            <div class="team-header">
              <input
                :value="team.name"
                @input="renameTeam(team.id, $event.target.value)"
                aria-label="Название команды"
              />
              <div class="team-color-picker" @click.stop>
                <button
                  class="team-color-trigger"
                  type="button"
                  :aria-label="`Цвет команды: ${team.name}`"
                  @click="toggleTeamColorPicker(team.id)"
                >
                  <span class="team-color-dot picker-dot" :style="{ backgroundColor: teamColorHex(team.color) }" aria-hidden="true"></span>
                </button>
                <div v-if="isTeamColorPickerOpen(team.id)" class="team-color-popover">
                  <button
                    v-for="color in teamColorOptions"
                    :key="`${team.id}-${color.value}`"
                    type="button"
                    class="team-color-option"
                    :title="color.label"
                    :aria-label="`Выбрать цвет: ${color.label}`"
                    @click="chooseTeamColor(team.id, color.value)"
                  >
                    <span class="team-color-dot picker-dot" :style="{ backgroundColor: color.swatch }" aria-hidden="true"></span>
                  </button>
                </div>
              </div>
              <button class="secondary" :disabled="teams.length <= 1" @click="removeTeam(team.id)">Удалить</button>
            </div>

            <div class="settings-events">
              <div v-for="player in team.players" :key="player.id" class="player-row">
                <input
                  :value="player.name"
                  @input="renamePlayer(team.id, player.id, $event.target.value)"
                  aria-label="Имя игрока"
                />
                <button class="secondary" :disabled="team.players.length <= 1" @click="removePlayer(team.id, player.id)">Удалить</button>
              </div>
            </div>

            <button class="secondary" @click="addPlayer(team.id)">Добавить игрока</button>
          </section>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

const teamColorOptions = [
  { value: 'lime', label: 'Салатовый', swatch: '#84cc16' },
  { value: 'orange', label: 'Оранжевый', swatch: '#f97316' },
  { value: 'black', label: 'Черный', swatch: '#111827' },
  { value: 'blue', label: 'Синий', swatch: '#2563eb' },
  { value: 'red', label: 'Красный', swatch: '#dc2626' },
  { value: 'yellow', label: 'Желтый', swatch: '#facc15' },
];
const defaultTeamColor = teamColorOptions[0].value;

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
const selectedRosterFilter = ref('all');
const showOnlyHighlights = ref(false);
const isTeamsOpen = ref(false);
const teams = ref(defaultTeams());
const selectedPlayerId = ref('');
const assigningPlayerEventId = ref('');
const openTeamColorPickerId = ref('');

const events = ref([]);
const currentTimeSec = ref(0);
const hasSyncedTime = ref(false);
const animatedEvent = ref(null);
const animatedEventRenderKey = ref(0);
const shareLinkStatus = ref('');

let syncInterval = null;
let animationTimeout = null;
let shareLinkStatusTimeout = null;
let playerShortcutTimeout = null;
let pendingPlayerShortcutTeamIndex = null;
let vkPlayer = null;

const eventGroups = [
  {
    id: 'shots',
    label: 'Броски',
    events: [
      { type: 'made_2pt', label: '2 очка', tone: 'positive', icon: '🏀' },
      { type: 'made_3pt', label: '3 очка', tone: 'positive', icon: '🎯' },
    ],
  },
    {
    id: 'misses',
    label: 'Промахи',
    events: [
      { type: 'missed_shot_2pt', label: '2 очка', tone: 'negative', icon: '❌' },
      { type: 'missed_shot_3pt', label: '3 очка', tone: 'negative', icon: '❌' },
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
      {type: 'steal', label: 'Перехват', tone: 'positive', icon: '🥷'},
      {type: 'block', label: 'Блок', tone: 'positive', icon: '💪'}
    ],
  },
  {
    id: 'rebounds',
    label: 'Подборы',
    events: [{ type: 'rebound', label: 'Подбор', tone: 'positive', icon: '👐' }],
  },
];

const eventTypes = eventGroups.flatMap((group) => group.events);
const eventShortcutKeys = ['q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '[', ']'];
const eventShortcutKeysRu = ['й', 'ц', 'у', 'к', 'е', 'н', 'г', 'ш', 'щ', 'з', 'х', 'ъ'];
const eventShortcutByType = Object.fromEntries(eventTypes.map((event, index) => [event.type, eventShortcutKeys[index] || '']));
const eventTypeByShortcut = eventTypes.reduce((acc, event, index) => {
  const shortcutEn = eventShortcutKeys[index];
  const shortcutRu = eventShortcutKeysRu[index];

  if (shortcutEn) {
    acc[shortcutEn] = event.type;
  }

  if (shortcutRu) {
    acc[shortcutRu] = event.type;
  }

  return acc;
}, {});
const storageKey = 'bball-statsman:v1';
function normalizeTeamColor(color) {
  const nextColor = String(color || '').trim();
  return teamColorOptions.some((option) => option.value === nextColor) ? nextColor : defaultTeamColor;
}

function teamColorHex(color) {
  return teamColorOptions.find((option) => option.value === color)?.swatch || teamColorOptions[0].swatch;
}


function defaultTeams() {
  return [
    {
      id: crypto.randomUUID(),
      name: 'Команда 1',
      color: defaultTeamColor,
      players: [
        {
          id: crypto.randomUUID(),
          name: 'Игрок 1',
        },
      ],
    },
  ];
}

function ensureTeamsStructure(inputTeams) {
  if (!Array.isArray(inputTeams) || !inputTeams.length) {
    return defaultTeams();
  }

  const normalizedTeams = inputTeams
    .filter((team) => team && typeof team === 'object')
    .map((team, teamIndex) => {
      const normalizedPlayers = Array.isArray(team.players)
        ? team.players
            .filter((player) => player && typeof player === 'object')
            .map((player, playerIndex) => ({
              id: player.id || crypto.randomUUID(),
              name: typeof player.name === 'string' && player.name.trim() ? player.name : `Игрок ${playerIndex + 1}`,
            }))
        : [];

      return {
        id: team.id || crypto.randomUUID(),
        name: typeof team.name === 'string' && team.name.trim() ? team.name : `Команда ${teamIndex + 1}`,
        color: normalizeTeamColor(team.color),
        players: normalizedPlayers.length
          ? normalizedPlayers
          : [
              {
                id: crypto.randomUUID(),
                name: 'Игрок 1',
              },
            ],
      };
    });

  return normalizedTeams.length ? normalizedTeams : defaultTeams();
}

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

const playerOptions = computed(() =>
  teams.value.flatMap((team, teamIndex) =>
    team.players.map((player, playerIndex) => ({
      id: player.id,
      playerName: player.name,
      teamName: team.name,
      shortcut: `${teamIndex + 1}${playerIndex + 1}`,
    })),
  ),
);
const selectedPlayerIndex = computed(() => playerOptions.value.findIndex((option) => option.id === selectedPlayerId.value));
const canSelectPreviousPlayer = computed(() => selectedPlayerIndex.value > 0);
const canSelectNextPlayer = computed(() => selectedPlayerIndex.value >= 0 && selectedPlayerIndex.value < playerOptions.value.length - 1);

const rosterFilterOptions = computed(() => {
  const options = [{ value: 'all', label: 'Все игроки' }];

  teams.value.forEach((team) => {
    options.push({
      value: `team:${team.id}`,
      label: `Команда: ${team.name}`,
    });

    team.players.forEach((player) => {
      options.push({
        value: `player:${player.id}`,
        label: `${team.name} · ${player.name}`,
      });
    });
  });

  return options;
});
const rosterFilterValues = computed(() => rosterFilterOptions.value.map((option) => option.value));
const selectedRosterFilterIndex = computed(() => rosterFilterValues.value.indexOf(selectedRosterFilter.value));
const canSelectPreviousRosterFilter = computed(() => selectedRosterFilterIndex.value > 0);
const canSelectNextRosterFilter = computed(() =>
  selectedRosterFilterIndex.value >= 0 && selectedRosterFilterIndex.value < rosterFilterValues.value.length - 1,
);

const eventsByTime = computed(() =>
  events.value
    .filter((event) => {
      if (selectedGameFilter.value === 'all') {
        return true;
      }

      const gameNumber = eventGameLabel(event.videoTimeSec);
      return String(gameNumber) === selectedGameFilter.value;
    })
    .filter((event) => matchRosterFilter(event))
    .sort((a, b) => a.videoTimeSec - b.videoTimeSec),
);

const filteredEvents = computed(() =>
  showOnlyHighlights.value ? eventsByTime.value.filter((event) => event.isHighlighted) : eventsByTime.value,
);

const summaryStats = computed(() => {
  let aggs = eventsByTime.value.reduce(
    (acc, event) => {
      if (event.type === 'made_2pt') {
        acc.points += 2;
        acc.made_2pt ++;
      }

      if (event.type === 'made_3pt') {
        acc.points += 3;
        acc.made_3pt ++;
      }

      if (event.type === 'missed_shot_2pt') {
        acc.missed_2pt += 1;
        acc.misses += 1;
      }

      if (event.type === 'missed_shot_3pt') {
        acc.missed_3pt += 1;
        acc.misses += 1;
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

      if (event.type === 'block') {
        acc.blocks += 1;
      }

      return acc;
    },
    { points: 0, rebounds: 0, blocks: 0, turnovers: 0, steals: 0, assists: 0, misses: 0, missed_2pt:0, missed_3pt: 0, made_2pt: 0, made_3pt: 0},
  )

  aggs.attempt_2pt = aggs.made_2pt + aggs.missed_2pt
  aggs.attempt_3pt = aggs.made_3pt + aggs.missed_3pt
  aggs.total_attempts = aggs.attempt_2pt + aggs.attempt_3pt
  aggs.percentage_2pt = aggs.attempt_2pt ? (aggs.made_2pt / aggs.attempt_2pt) * 100 : 0
  aggs.percentage_3pt = aggs.attempt_3pt ? (aggs.made_3pt / aggs.attempt_3pt) * 100 : 0
  aggs.percentage_all = aggs.total_attempts ? ((aggs.made_2pt + aggs.made_3pt) / aggs.total_attempts) * 100 : 0

  return aggs
}
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
        selectedRosterFilter: selectedRosterFilter.value,
        showOnlyHighlights: showOnlyHighlights.value,
        teams: teams.value,
        selectedPlayerId: selectedPlayerId.value,
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
  const storedGameFilter = String(videoState?.settings?.selectedGameFilter || 'all');
  selectedGameFilter.value = storedGameFilter === 'all' || availableGameIds.includes(storedGameFilter) ? storedGameFilter : 'all';
  showOnlyHighlights.value = Boolean(videoState?.settings?.showOnlyHighlights);
  teams.value = ensureTeamsStructure(videoState?.settings?.teams);

  const rosterValues = rosterFilterOptions.value.map((option) => option.value);
  const storedRosterFilter = String(videoState?.settings?.selectedRosterFilter || 'all');
  selectedRosterFilter.value = rosterValues.includes(storedRosterFilter) ? storedRosterFilter : 'all';

  const firstPlayerId = playerOptions.value[0]?.id || '';
  const storedPlayerId = String(videoState?.settings?.selectedPlayerId || '');
  selectedPlayerId.value = playerOptions.value.some((player) => player.id === storedPlayerId) ? storedPlayerId : firstPlayerId;
}

function resetVideoState() {
  events.value = [];
  gameRanges.value = [];
  selectedGameFilter.value = 'all';
  selectedRosterFilter.value = 'all';
  showOnlyHighlights.value = false;
  groupVisibility.value = defaultGroupVisibility();
  eventVisibility.value = defaultEventVisibility();
  teams.value = defaultTeams();
  selectedPlayerId.value = teams.value[0]?.players?.[0]?.id || '';
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

function getVideoQueryParamValue() {
  return new URLSearchParams(window.location.search).get('video') || '';
}

function buildShareUrl(video) {
  const shareUrl = new URL(window.location.href);
  shareUrl.searchParams.set('video', video);
  return shareUrl.toString();
}

function syncVideoQueryParam(video) {
  const nextUrl = new URL(window.location.href);

  if (video) {
    nextUrl.searchParams.set('video', video);
  } else {
    nextUrl.searchParams.delete('video');
  }

  window.history.replaceState({}, '', nextUrl);
}

function setShareStatus(message) {
  shareLinkStatus.value = message;

  if (shareLinkStatusTimeout) {
    clearTimeout(shareLinkStatusTimeout);
  }

  shareLinkStatusTimeout = setTimeout(() => {
    shareLinkStatus.value = '';
  }, 2200);
}

async function copyShareLink() {
  if (!activeVideoUrl.value) {
    return;
  }

  const shareUrl = buildShareUrl(activeVideoUrl.value);

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(shareUrl);
      setShareStatus('Ссылка скопирована.');
      return;
    }
  } catch {
    // fallback below
  }

  const textarea = document.createElement('textarea');
  textarea.value = shareUrl;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'absolute';
  textarea.style.left = '-9999px';
  document.body.append(textarea);
  textarea.select();

  const isCopied = document.execCommand('copy');
  textarea.remove();
  setShareStatus(isCopied ? 'Ссылка скопирована.' : 'Не удалось скопировать ссылку.');
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

  if (!rosterFilterValues.value.includes(selectedRosterFilter.value)) {
    selectedRosterFilter.value = 'all';
  }
  isSessionStarted.value = true;
  syncVideoQueryParam(normalizedUrl);
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
  shareLinkStatus.value = '';
  syncVideoQueryParam('');
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
    playerId: selectedPlayerId.value,
    isHighlighted: false,
  });
}

function eventShortcutLabel(type) {
  const shortcut = eventShortcutByType[type];
  return shortcut ? `[${shortcut}]` : '';
}


function selectPreviousPlayer() {
  if (!canSelectPreviousPlayer.value) {
    return;
  }

  selectedPlayerId.value = playerOptions.value[selectedPlayerIndex.value - 1].id;
}

function selectNextPlayer() {
  if (!canSelectNextPlayer.value) {
    return;
  }

  selectedPlayerId.value = playerOptions.value[selectedPlayerIndex.value + 1].id;
}

function addTeam() {
  const teamNumber = teams.value.length + 1;
  teams.value.push({
    id: crypto.randomUUID(),
    name: `Команда ${teamNumber}`,
    color: defaultTeamColor,
    players: [
      {
        id: crypto.randomUUID(),
        name: 'Игрок 1',
      },
    ],
  });
}

function renameTeam(teamId, nextName) {
  teams.value = teams.value.map((team) => (team.id === teamId ? { ...team, name: nextName } : team));
}

function setTeamColor(teamId, color) {
  teams.value = teams.value.map((team) =>
    team.id === teamId
      ? {
          ...team,
          color: normalizeTeamColor(color),
        }
      : team,
  );
}
function isTeamColorPickerOpen(teamId) {
  return openTeamColorPickerId.value === teamId;
}

function toggleTeamColorPicker(teamId) {
  openTeamColorPickerId.value = openTeamColorPickerId.value === teamId ? '' : teamId;
}

function chooseTeamColor(teamId, color) {
  setTeamColor(teamId, color);
  openTeamColorPickerId.value = '';
}

function closeTeamColorPicker() {
  openTeamColorPickerId.value = '';
}

function resetPlayerShortcutState() {
  if (playerShortcutTimeout) {
    clearTimeout(playerShortcutTimeout);
    playerShortcutTimeout = null;
  }

  pendingPlayerShortcutTeamIndex = null;
}

function handlePlayerShortcutKeydown(event) {
  if (event.ctrlKey || event.metaKey || event.altKey || event.isComposing) {
    return;
  }

  const target = event.target;
  if (target?.closest?.('input, textarea, select, [contenteditable="true"]')) {
    return;
  }

  const key = String(event.key || '').toLowerCase();
  const eventType = eventTypeByShortcut[key];
  if (eventType) {
    addEvent(eventType);
    event.preventDefault();
    return;
  }

  const shortcutDigit = Number(event.key);
  if (!Number.isInteger(shortcutDigit) || shortcutDigit < 1 || shortcutDigit > 9) {
    return;
  }

  if (pendingPlayerShortcutTeamIndex === null) {
    if (shortcutDigit > teams.value.length) {
      resetPlayerShortcutState();
      return;
    }

    pendingPlayerShortcutTeamIndex = shortcutDigit - 1;
    if (playerShortcutTimeout) {
      clearTimeout(playerShortcutTimeout);
    }

    playerShortcutTimeout = setTimeout(() => {
      resetPlayerShortcutState();
    }, 1000);
    return;
  }

  const team = teams.value[pendingPlayerShortcutTeamIndex];
  resetPlayerShortcutState();
  if (!team || shortcutDigit > team.players.length) {
    return;
  }

  selectedPlayerId.value = team.players[shortcutDigit - 1].id;
  event.preventDefault();
}

function handleDocumentClick(event) {
  if (event.target?.closest?.('.team-color-picker')) {
    return;
  }

  closeTeamColorPicker();
}


function removeTeam(teamId) {
  if (teams.value.length <= 1) {
    return;
  }

  teams.value = teams.value.filter((team) => team.id !== teamId);

  if (openTeamColorPickerId.value === teamId) {
    closeTeamColorPicker();
  }
}

function addPlayer(teamId) {
  teams.value = teams.value.map((team) => {
    if (team.id !== teamId) {
      return team;
    }

    return {
      ...team,
      players: [
        ...team.players,
        {
          id: crypto.randomUUID(),
          name: `Игрок ${team.players.length + 1}`,
        },
      ],
    };
  });
}

function renamePlayer(teamId, playerId, nextName) {
  teams.value = teams.value.map((team) => {
    if (team.id !== teamId) {
      return team;
    }

    return {
      ...team,
      players: team.players.map((player) => (player.id === playerId ? { ...player, name: nextName } : player)),
    };
  });
}

function removePlayer(teamId, playerId) {
  teams.value = teams.value.map((team) => {
    if (team.id !== teamId || team.players.length <= 1) {
      return team;
    }

    return {
      ...team,
      players: team.players.filter((player) => player.id !== playerId),
    };
  });
}

function eventPlayerMeta(event) {
  const fallback = {
    label: 'Неизвестный игрок',
    playerName: 'Неизвестный игрок',
    teamColor: teamColorHex(defaultTeamColor),
    isUnknown: true,
  };

  const playerId = event?.playerId;
  if (!playerId) {
    return fallback;
  }

  for (const team of teams.value) {
    const player = team.players.find((candidate) => candidate.id === playerId);
    if (player) {
      return {
        label: `${team.name} · ${player.name}`,
        playerName: player.name,
        teamColor: teamColorHex(team.color),
        isUnknown: false,
      };
    }
  }

  return fallback;
}

function eventPlayerLabel(event) {
  return eventPlayerMeta(event).label;
}

function eventPlayerName(event) {
  return eventPlayerMeta(event).playerName;
}

function eventPlayerTeamColor(event) {
  return eventPlayerMeta(event).teamColor;
}

function isUnknownEventPlayer(event) {
  return eventPlayerMeta(event).isUnknown;
}

function startAssignEventPlayer(eventId) {
  assigningPlayerEventId.value = eventId;
}

function cancelAssignEventPlayer() {
  assigningPlayerEventId.value = '';
}

function assignEventPlayer(eventId, playerId) {
  if (!playerId) {
    cancelAssignEventPlayer();
    return;
  }

  events.value = events.value.map((event) => {
    if (event.id !== eventId) {
      return event;
    }

    return {
      ...event,
      playerId,
    };
  });

  cancelAssignEventPlayer();
}

function toggleEventHighlight(eventId) {
  events.value = events.value.map((event) => {
    if (event.id !== eventId) {
      return event;
    }

    return {
      ...event,
      isHighlighted: !event.isHighlighted,
    };
  });
}

function removeEvent(eventId) {
  events.value = events.value.filter((event) => event.id !== eventId);

  if (assigningPlayerEventId.value === eventId) {
    cancelAssignEventPlayer();
  }
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


function findTeamByPlayerId(playerId) {
  return teams.value.find((team) => team.players.some((player) => player.id === playerId)) || null;
}

function matchRosterFilter(event) {
  if (selectedRosterFilter.value === 'all') {
    return true;
  }

  const [scope, id] = selectedRosterFilter.value.split(':');

  if (scope === 'team') {
    return findTeamByPlayerId(event.playerId)?.id === id;
  }

  if (scope === 'player') {
    return event.playerId === id;
  }

  return true;
}

function selectPreviousRosterFilter() {
  if (!canSelectPreviousRosterFilter.value) {
    return;
  }

  selectedRosterFilter.value = rosterFilterValues.value[selectedRosterFilterIndex.value - 1];
}

function selectNextRosterFilter() {
  if (!canSelectNextRosterFilter.value) {
    return;
  }

  selectedRosterFilter.value = rosterFilterValues.value[selectedRosterFilterIndex.value + 1];
}

function clearEvents() {
  const isConfirmed = window.confirm('Вы точно хотите удалить все события?');

  if (!isConfirmed) {
    return;
  }

  events.value = [];
  cancelAssignEventPlayer();
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
  [events, gameRanges, selectedGameFilter, selectedRosterFilter, showOnlyHighlights, groupVisibility, eventVisibility, teams, selectedPlayerId],
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



watch(
  teams,
  () => {
    if (!playerOptions.value.some((option) => option.id === selectedPlayerId.value)) {
      selectedPlayerId.value = playerOptions.value[0]?.id || '';
    }

    if (!rosterFilterValues.value.includes(selectedRosterFilter.value)) {
      selectedRosterFilter.value = 'all';
    }
  },
  { deep: true },
);

onMounted(() => {
  window.addEventListener('message', handlePlayerMessage);
  document.addEventListener('click', handleDocumentClick);
  document.addEventListener('keydown', handlePlayerShortcutKeydown);

  if (!selectedPlayerId.value) {
    selectedPlayerId.value = playerOptions.value[0]?.id || '';
  }

  const initialVideoParam = getVideoQueryParamValue();
  if (initialVideoParam) {
    videoUrl.value = initialVideoParam;
    startSession(initialVideoParam);
  }
});

onBeforeUnmount(() => {
  stopSync();
  vkPlayer = null;
  if (animationTimeout) {
    clearTimeout(animationTimeout);
    animationTimeout = null;
  }
  if (shareLinkStatusTimeout) {
    clearTimeout(shareLinkStatusTimeout);
    shareLinkStatusTimeout = null;
  }
  resetPlayerShortcutState();
  animatedEvent.value = null;
  window.removeEventListener('message', handlePlayerMessage);
  document.removeEventListener('click', handleDocumentClick);
  document.removeEventListener('keydown', handlePlayerShortcutKeydown);
});
</script>
