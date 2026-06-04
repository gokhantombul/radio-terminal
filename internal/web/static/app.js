document.addEventListener('DOMContentLoaded', () => {
    const DEFAULT_THEME = 'shell-glass';
    const THEMES = new Set(['shell-glass', 'premium-radio', 'compact-dashboard', 'neon-live', 'midnight', 'sakura', 'retro-crt', 'brutalist', 'theme-car', 'theme-win95', 'winamp-classic', 'besiktas-bjk']);

    const stationList = document.getElementById('station-list');
    const searchInput = document.getElementById('search-input');
    const filterRow = document.getElementById('filter-row');
    const nowPlaying = document.getElementById('now-playing');
    const stationNameDisplay = document.getElementById('current-station-name');
    const songTitleDisplay = document.getElementById('current-song-title');
    const stopBtn = document.getElementById('stop-btn');
    const prevBtn = document.getElementById('prev-btn');
    const nextBtn = document.getElementById('next-btn');
    const shuffleBtn = document.getElementById('shuffle-btn');
    const currentFavoriteBtn = document.getElementById('current-favorite-btn');
    const currentFavoriteLabel = document.getElementById('current-favorite-label');
    const muteBtn = document.getElementById('mute-btn');
    const muteLabel = document.getElementById('mute-label');
    const recordBtn = document.getElementById('record-btn');
    const recordingPill = document.getElementById('recording-pill');
    const elapsedPill = document.getElementById('elapsed-time');
    const volumeSlider = document.getElementById('volume-slider');
    const volumeValue = document.getElementById('volume-value');
    const equalizer = document.getElementById('equalizer');
    const langSelect = document.getElementById('language-select');
    const themeSelect = document.getElementById('theme-select');
    const systemBtn = document.getElementById('system-btn');
    const systemModal = document.getElementById('system-modal');
    const closeModal = document.getElementById('close-modal');
    const systemStats = document.getElementById('system-stats');
    const toastRegion = document.getElementById('toast-region');

    let allStations = [];
    let currentStationId = null;
    let isRecording = false;
    let isMuted = false;
    let stationsLoaded = false;
    let locales = {};
    let activeFilter = { type: 'all', value: '' };

    let elapsedBase = null;
    let elapsedBaseAt = null;
    let elapsedTickId = null;

    function startElapsedTick(serverSeconds) {
        elapsedBase = serverSeconds;
        elapsedBaseAt = Date.now();
        if (elapsedTickId) clearInterval(elapsedTickId);
        elapsedTickId = setInterval(() => {
            const secs = elapsedBase + Math.floor((Date.now() - elapsedBaseAt) / 1000);
            const m = Math.floor(secs / 60);
            const s = secs % 60;
            elapsedPill.textContent = `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
        }, 1000);
    }

    function stopElapsedTick() {
        if (elapsedTickId) { clearInterval(elapsedTickId); elapsedTickId = null; }
        elapsedBase = null;
        elapsedBaseAt = null;
    }

    function t(key, fallback) {
        return locales[key] || fallback;
    }

    function setTheme(theme) {
        const selectedTheme = THEMES.has(theme) ? theme : DEFAULT_THEME;
        document.body.dataset.theme = selectedTheme;
        themeSelect.value = selectedTheme;
        localStorage.setItem('radio-web-theme', selectedTheme);
    }

    function initTheme() {
        const savedTheme = localStorage.getItem('radio-web-theme');
        const initialTheme = savedTheme || DEFAULT_THEME;
        setTheme(initialTheme);
    }

    function applyTranslations() {
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            if (locales[key]) el.textContent = locales[key];
        });

        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.getAttribute('data-i18n-placeholder');
            if (locales[key]) el.setAttribute('placeholder', locales[key]);
        });

        document.querySelectorAll('[data-i18n-title]').forEach(el => {
            const key = el.getAttribute('data-i18n-title');
            if (locales[key]) el.setAttribute('title', locales[key]);
        });

        document.querySelectorAll('[data-i18n-tooltip]').forEach(el => {
            const key = el.getAttribute('data-i18n-tooltip');
            if (locales[key]) el.dataset.tooltip = locales[key];
        });

        document.querySelectorAll('[data-i18n-aria-label]').forEach(el => {
            const key = el.getAttribute('data-i18n-aria-label');
            if (locales[key]) el.setAttribute('aria-label', locales[key]);
        });
    }

    async function fetchLocalization() {
        try {
            const langRes = await fetch('/api/language');
            if (!langRes.ok) throw new Error('Language API error');
            const langData = await langRes.json();

            if (langData.available) {
                langSelect.innerHTML = '';
                const sortedLangs = Object.entries(langData.available).sort((a, b) => a[1].localeCompare(b[1]));

                for (const [code, name] of sortedLangs) {
                    const opt = document.createElement('option');
                    opt.value = code;
                    opt.textContent = name || code.toUpperCase();
                    if (code === langData.current) opt.selected = true;
                    langSelect.appendChild(opt);
                }
            }

            const locRes = await fetch('/api/locales');
            if (!locRes.ok) throw new Error('Locales API error');
            locales = await locRes.json();
            applyTranslations();
            updateMuteUi(isMuted);
            updateCurrentFavoriteUi();
            buildFilters();
            if (stationsLoaded) {
                renderStations(getVisibleStations());
            }
        } catch (error) {
            console.error('Localization error:', error);
            if (langSelect.options.length === 0) {
                langSelect.innerHTML = '<option value="en">English</option><option value="tr">Türkçe</option>';
            }
        }
    }

    function showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type === 'error' ? 'error' : ''}`;
        toast.textContent = message;
        toastRegion.appendChild(toast);

        window.setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateY(8px)';
            window.setTimeout(() => toast.remove(), 180);
        }, 3200);
    }

    function renderState(message, isError = false) {
        stationList.innerHTML = '';
        const state = document.createElement('div');
        state.className = 'state-container';
        state.innerHTML = isError
            ? `<p style="color: var(--danger);">${message}</p>`
            : `<div class="spinner"></div><p>${message}</p>`;
        stationList.appendChild(state);
    }

    async function fetchStations() {
        renderState(t('web_stations_loading', 'Loading stations...'));
        try {
            const response = await fetch('/api/stations');
            if (!response.ok) throw new Error('Stations API error');
            allStations = await response.json();
            stationsLoaded = true;
            buildFilters();
            renderStations(getVisibleStations());
        } catch (error) {
            console.error('Error fetching stations:', error);
            renderState(t('msg_error', 'Error'), true);
        }
    }

    function mostCommonValues(field, limit) {
        const counts = new Map();
        allStations.forEach(station => {
            const value = (station[field] || '').trim();
            if (!value || value === '-') return;
            counts.set(value, (counts.get(value) || 0) + 1);
        });

        return Array.from(counts.entries())
            .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
            .slice(0, limit)
            .map(([value]) => value);
    }

    function createFilterChip(type, value, label) {
        const chip = document.createElement('button');
        chip.type = 'button';
        chip.className = `filter-chip ${activeFilter.type === type && activeFilter.value === value ? 'active' : ''}`;
        chip.dataset.filterType = type;
        chip.dataset.filterValue = value;
        chip.textContent = label;
        chip.addEventListener('click', () => {
            activeFilter = { type, value };
            buildFilters();
            renderStations(getVisibleStations());
        });
        return chip;
    }

    function buildFilters() {
        if (!filterRow) return;
        filterRow.innerHTML = '';
        filterRow.appendChild(createFilterChip('all', '', t('web_filter_all', 'All')));
        filterRow.appendChild(createFilterChip('favorites', '', t('web_filter_favorites', 'Favorites')));

        mostCommonValues('genre', 3).forEach(genre => {
            filterRow.appendChild(createFilterChip('genre', genre, `${t('genre', 'Genre')}: ${genre}`));
        });

        mostCommonValues('country', 2).forEach(country => {
            filterRow.appendChild(createFilterChip('country', country, `${t('country', 'Country')}: ${country}`));
        });
    }

    function getVisibleStations() {
        const term = searchInput.value.trim().toLowerCase();

        return allStations.filter(station => {
            const stationText = [
                station.name,
                station.genre,
                station.country
            ].filter(Boolean).join(' ').toLowerCase();

            const matchesSearch = !term || stationText.includes(term);
            const matchesFilter =
                activeFilter.type === 'all' ||
                (activeFilter.type === 'favorites' && station.is_favorite) ||
                (activeFilter.type === 'genre' && station.genre === activeFilter.value) ||
                (activeFilter.type === 'country' && station.country === activeFilter.value);

            return matchesSearch && matchesFilter;
        });
    }

    function getCurrentStation() {
        return allStations.find(station => station.id === currentStationId) || null;
    }

    function updatePlaybackControlsUi() {
        const hasVisibleStations = getVisibleStations().length > 0;
        prevBtn.disabled = !hasVisibleStations;
        nextBtn.disabled = !hasVisibleStations;
        shuffleBtn.disabled = !hasVisibleStations;
        updateCurrentFavoriteUi();
    }

    function updateCurrentFavoriteUi() {
        const currentStation = getCurrentStation();
        const isFavorite = Boolean(currentStation && currentStation.is_favorite);
        currentFavoriteBtn.disabled = !currentStationId;
        currentFavoriteBtn.classList.toggle('is-favorite', isFavorite);
        currentFavoriteBtn.setAttribute('aria-pressed', String(isFavorite));

        const label = isFavorite
            ? t('web_unfavorite', 'Unfavorite')
            : t('web_favorite', 'Favorite');
        currentFavoriteLabel.textContent = label;
        currentFavoriteBtn.dataset.tooltip = label;
        currentFavoriteBtn.setAttribute('aria-label', label);
    }

    function starIcon(isFavorite) {
        const fill = isFavorite ? 'currentColor' : 'none';
        return `
            <svg viewBox="0 0 24 24" role="img" aria-hidden="true">
                <path fill="${fill}" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" d="m12 3.6 2.6 5.2 5.8.8-4.2 4.1 1 5.8L12 16.8l-5.2 2.7 1-5.8-4.2-4.1 5.8-.8L12 3.6Z"></path>
            </svg>
        `;
    }

    function renderStations(stations) {
        stationList.innerHTML = '';

        if (stations.length === 0) {
            const state = document.createElement('div');
            state.className = 'state-container';
            state.innerHTML = `<p>${t('web_no_results', 'No results found.')}</p>`;
            stationList.appendChild(state);
            updatePlaybackControlsUi();
            return;
        }

        stations.forEach(station => {
            const card = document.createElement('article');
            card.className = `station-card ${currentStationId === station.id ? 'active' : ''}`;
            card.tabIndex = 0;
            card.setAttribute('role', 'button');
            card.setAttribute('aria-label', `${t('web_play', 'Play')} ${station.name}`);

            const initial = (station.name || '?').trim().charAt(0).toUpperCase() || '?';
            const genre = station.genre || t('web_unknown_genre', 'Unknown genre');
            const country = station.country || t('web_unknown_country', 'Unknown country');
            const isActive = currentStationId === station.id;

            card.innerHTML = `
                <div class="card-top">
                    <div class="station-avatar"></div>
                    <button class="fav-badge ${station.is_favorite ? 'is-favorite' : ''}" type="button" aria-label="${t('web_toggle_favorite', 'Toggle favorite')}">
                        ${starIcon(station.is_favorite)}
                    </button>
                </div>
                <div class="station-main">
                    <div class="station-name-card"></div>
                    <div class="station-meta">
                        <span class="meta-pill"></span>
                        <span class="meta-pill"></span>
                    </div>
                </div>
                <div class="card-action">
                    <span class="play-indicator">${isActive ? t('web_playing_short', 'Playing') : t('web_live', 'Live Broadcast')}</span>
                    <span class="play-arrow" aria-hidden="true">
                        <svg viewBox="0 0 24 24" role="img"><path d="M8 5.5v13l10-6.5-10-6.5Z"/></svg>
                    </span>
                </div>
            `;

            card.querySelector('.station-avatar').textContent = initial;
            card.querySelector('.station-name-card').textContent = station.name;
            const metaPills = card.querySelectorAll('.meta-pill');
            metaPills[0].textContent = genre;
            metaPills[1].textContent = country;

            card.querySelector('.fav-badge').addEventListener('click', event => {
                event.stopPropagation();
                toggleFavorite(station.id);
            });

            card.addEventListener('click', () => playStation(station.id));
            card.addEventListener('keydown', event => {
                if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    playStation(station.id);
                }
            });

            stationList.appendChild(card);
        });

        updatePlaybackControlsUi();
    }

    async function toggleFavorite(id) {
        try {
            const response = await fetch(`/api/favorite/${encodeURIComponent(id)}`, { method: 'POST' });
            if (!response.ok) throw new Error('Favorite API error');
            const data = await response.json();
            const station = allStations.find(s => s.id === id);
            if (station) {
                station.is_favorite = data.is_favorite;
                showToast(
                    data.is_favorite
                        ? t('web_favorite_added', 'Added to favorites')
                        : t('web_favorite_removed', 'Removed from favorites')
                );
            }
            buildFilters();
            renderStations(getVisibleStations());
            updateCurrentFavoriteUi();
            return data.is_favorite;
        } catch (error) {
            console.error('Error toggling favorite:', error);
            showToast(t('msg_error', 'Error'), 'error');
            return null;
        }
    }

    function pickVisibleStation(offset) {
        const visibleStations = getVisibleStations();
        if (visibleStations.length === 0) return null;

        const currentIndex = visibleStations.findIndex(station => station.id === currentStationId);
        if (currentIndex === -1) {
            return offset >= 0 ? visibleStations[0] : visibleStations[visibleStations.length - 1];
        }

        const nextIndex = (currentIndex + offset + visibleStations.length) % visibleStations.length;
        return visibleStations[nextIndex];
    }

    async function playAdjacent(offset) {
        const station = pickVisibleStation(offset);
        if (!station) {
            showToast(t('web_no_results', 'No results found.'), 'error');
            return;
        }
        await playStation(station.id);
    }

    async function shuffleVisibleStation() {
        const visibleStations = getVisibleStations();
        if (visibleStations.length === 0) {
            showToast(t('web_no_results', 'No results found.'), 'error');
            return;
        }

        const candidates = visibleStations.length > 1
            ? visibleStations.filter(station => station.id !== currentStationId)
            : visibleStations;
        const station = candidates[Math.floor(Math.random() * candidates.length)];
        await playStation(station.id);
    }

    async function toggleCurrentFavorite() {
        if (!currentStationId) {
            showToast(t('msg_no_playing_station', 'No station currently playing.'), 'error');
            return;
        }
        await toggleFavorite(currentStationId);
    }

    async function playStation(id) {
        try {
            currentStationId = id;
            renderStations(getVisibleStations());
            stationNameDisplay.textContent = t('web_connecting', 'Connecting...');
            songTitleDisplay.textContent = '';
            nowPlaying.classList.remove('hidden');

            const response = await fetch(`/api/play/${encodeURIComponent(id)}`, { method: 'POST' });
            if (!response.ok) throw new Error('Play API error');
            await updateStatus();
        } catch (error) {
            console.error('Error playing station:', error);
            showToast(t('msg_error', 'Error'), 'error');
        }
    }

    async function stopPlayback() {
        try {
            const response = await fetch('/api/stop', { method: 'POST' });
            if (!response.ok) throw new Error('Stop API error');
            currentStationId = null;
            await updateStatus();
            renderStations(getVisibleStations());
            showToast(t('msg_stop_playing', 'Playback stopped.'));
        } catch (error) {
            console.error('Error stopping playback:', error);
            showToast(t('msg_error', 'Error'), 'error');
        }
    }

    function updateRecordingUi(recording) {
        isRecording = recording;
        recordBtn.classList.toggle('recording', recording);
        recordingPill.classList.toggle('hidden', !recording);
    }

    function updateMuteUi(muted) {
        isMuted = muted;
        muteBtn.classList.toggle('muted', muted);
        muteBtn.setAttribute('aria-pressed', String(muted));

        const label = muted
            ? t('web_unmute', 'Unmute')
            : t('web_mute', 'Mute');
        muteLabel.textContent = label;
        muteBtn.dataset.tooltip = label;
        muteBtn.setAttribute('aria-label', label);
    }

    async function updateStatus() {
        try {
            const response = await fetch('/api/status');
            if (!response.ok) throw new Error('Status API error');
            const status = await response.json();
            updateMuteUi(Boolean(status.is_muted));

            if (status.is_playing) {
                nowPlaying.classList.remove('hidden');

                if (status.current_station) {
                    stationNameDisplay.textContent = status.current_station.name;
                    const station = allStations.find(s => s.id === status.current_station.id);
                    if (station) {
                        station.is_favorite = Boolean(status.current_station.is_favorite);
                    }
                    if (currentStationId !== status.current_station.id) {
                        currentStationId = status.current_station.id;
                        renderStations(getVisibleStations());
                    }
                    updateCurrentFavoriteUi();
                }

                songTitleDisplay.textContent =
                    status.current_song && status.current_song !== '-'
                        ? status.current_song
                        : t('web_live', 'Live Broadcast');

                equalizer.classList.remove('paused');
                if (document.activeElement !== volumeSlider) {
                    volumeSlider.value = status.volume;
                    volumeValue.textContent = `${status.volume}%`;
                }
                updateRecordingUi(Boolean(status.is_recording));

                if (status.elapsed_seconds != null) {
                    elapsedPill.classList.remove('hidden');
                    if (elapsedBase === null) {
                        startElapsedTick(status.elapsed_seconds);
                    } else {
                        elapsedBase = status.elapsed_seconds;
                        elapsedBaseAt = Date.now();
                    }
                } else {
                    stopElapsedTick();
                    elapsedPill.classList.add('hidden');
                }
            } else {
                nowPlaying.classList.add('hidden');
                const hadCurrentStation = Boolean(currentStationId);
                currentStationId = null;
                equalizer.classList.add('paused');
                updateRecordingUi(false);
                updateCurrentFavoriteUi();
                if (hadCurrentStation) {
                    renderStations(getVisibleStations());
                }
                stopElapsedTick();
                elapsedPill.classList.add('hidden');
            }
        } catch (error) {
            console.error('Error updating status:', error);
        }
    }

    async function toggleRecording() {
        try {
            const endpoint = isRecording ? '/api/record/stop' : '/api/record/start';
            const response = await fetch(endpoint, { method: 'POST' });
            if (!response.ok) throw new Error('Recording API error');
            const result = await response.json();
            showToast(result.message || t('web_recording_updated', 'Recording updated.'));
            await updateStatus();
        } catch (error) {
            console.error('Recording error:', error);
            showToast(t('msg_error', 'Error'), 'error');
        }
    }

    async function setVolume(level) {
        try {
            const response = await fetch(`/api/volume/${level}`, { method: 'POST' });
            if (!response.ok) throw new Error('Volume API error');
            const result = await response.json();
            updateMuteUi(Boolean(result.is_muted));
        } catch (error) {
            console.error('Error setting volume:', error);
            showToast(t('msg_error', 'Error'), 'error');
        }
    }

    async function toggleMute() {
        try {
            const response = await fetch(`/api/mute/${!isMuted}`, { method: 'POST' });
            if (!response.ok) throw new Error('Mute API error');
            const result = await response.json();
            updateMuteUi(Boolean(result.is_muted));
            showToast(result.is_muted ? t('msg_muted', 'Muted.') : t('msg_unmuted', 'Sound restored.'));
        } catch (error) {
            console.error('Mute error:', error);
            showToast(t('msg_error', 'Error'), 'error');
        }
    }

    async function showSystemModal() {
        try {
            systemStats.innerHTML = '<div class="spinner" style="margin: 0 auto;"></div>';
            systemModal.classList.remove('hidden');
            const response = await fetch('/api/system');
            if (!response.ok) throw new Error('System API error');
            const data = await response.json();
            systemStats.innerHTML = `
                <div class="system-stat"><span>OS</span><strong>${data.os}</strong></div>
                <div class="system-stat"><span>Python</span><strong>${data.python_version}</strong></div>
                <div class="system-stat"><span>RAM</span><strong>${data.memory_usage_mb} MB</strong></div>
                <div class="system-stat"><span>CPU</span><strong>${data.cpu_percent}%</strong></div>
            `;
        } catch (error) {
            console.error('System modal error:', error);
            systemStats.innerHTML = `<p style="color:var(--danger);">${t('msg_error', 'Error')}</p>`;
        }
    }

    initTheme();

    themeSelect.addEventListener('change', event => setTheme(event.target.value));

    langSelect.addEventListener('change', async event => {
        try {
            const response = await fetch(`/api/language/${event.target.value}`, { method: 'POST' });
            if (!response.ok) throw new Error('Language API error');
            await fetchLocalization();
        } catch (error) {
            console.error('Error changing language:', error);
            showToast(t('msg_error', 'Error'), 'error');
        }
    });

    searchInput.addEventListener('input', () => renderStations(getVisibleStations()));
    prevBtn.addEventListener('click', () => playAdjacent(-1));
    nextBtn.addEventListener('click', () => playAdjacent(1));
    shuffleBtn.addEventListener('click', shuffleVisibleStation);
    currentFavoriteBtn.addEventListener('click', toggleCurrentFavorite);
    stopBtn.addEventListener('click', stopPlayback);
    muteBtn.addEventListener('click', toggleMute);
    recordBtn.addEventListener('click', toggleRecording);
    volumeSlider.addEventListener('input', event => {
        volumeValue.textContent = `${event.target.value}%`;
    });
    volumeSlider.addEventListener('change', event => setVolume(event.target.value));
    systemBtn.addEventListener('click', showSystemModal);
    closeModal.addEventListener('click', () => systemModal.classList.add('hidden'));
    systemModal.addEventListener('click', event => {
        if (event.target === systemModal) systemModal.classList.add('hidden');
    });
    document.addEventListener('keydown', event => {
        if (event.key === 'Escape') systemModal.classList.add('hidden');
    });

    fetchLocalization();
    fetchStations();
    updateStatus();
    window.setInterval(updateStatus, 2000);
});
