/* 原生 JavaScript 页面：通过同源 fetch 调用 Gin API，不需要 Node 构建。 */
(() => {
  'use strict';

  const $ = (id) => document.getElementById(id);
  const PAGE_SIZE = 10;
  const SESSION_KEY = 'yingliu.session.v1';
  const FEEDS = {
    latest: { title: '发现新鲜视角', description: '从最新发布开始，找到让你停留的片刻。', tag: '最近发布' },
    hot: { title: '好内容，正在被看见', description: '看看最近 24 小时，大家喜欢的那些作品。', tag: '热门精选' },
    following: { title: '喜欢的创作，继续发生', description: '你关注的创作者，新作品都在这里。', tag: '关注动态' },
  };
  const state = {
    feed: 'latest', videos: [], videoStore: new Map(), page: 0, cursor: '',
    hasMore: false, total: 0, hotLimit: 20, degraded: false, search: '',
    loading: false, feedError: '', feedVersion: 0, feedController: null,
    ranking: [], rankingLoading: false, rankingError: '', rankingDegraded: false,
    rankingVersion: 0, rankingController: null, refreshTimer: null,
    token: '', user: null, authMode: 'login', authPending: false, afterLogin: '', openVersion: 0,
    publishing: false, loggingOut: false, currentVideo: null, watchVersion: 0,
    liked: null, following: null, statusLoading: false,
    likeBusy: false, followBusy: false, comments: [], commentsLoading: false,
    commentsError: '', commentsController: null, commentBusy: false, deleting: new Set(),
  };

  class ApiError extends Error {
    constructor(message, status = 0) { super(message); this.name = 'ApiError'; this.status = status; }
  }

  function element(tag, className = '', text) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function icon(name) {
    const node = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    node.classList.add('icon');
    node.setAttribute('aria-hidden', 'true');
    const use = document.createElementNS('http://www.w3.org/2000/svg', 'use');
    use.setAttribute('href', `#i-${name}`);
    node.append(use);
    return node;
  }

  function button(text, action, className = 'button button-primary', iconName) {
    const node = element('button', className);
    node.type = 'button';
    node.dataset.action = action;
    if (iconName) node.append(icon(iconName));
    node.append(element('span', '', text));
    return node;
  }

  function positiveID(value) {
    const id = Number(value);
    return Number.isSafeInteger(id) && id > 0 ? id : 0;
  }

  function safeURL(value) {
    if (!value || typeof value !== 'string') return '';
    try {
      const url = new URL(value);
      return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password ? url.href : '';
    } catch { return ''; }
  }

  function isExampleURL(value) {
    try { return /(^|\.)example\.(com|org|net)$/.test(new URL(value).hostname); }
    catch { return false; }
  }

  function normalizeVideo(value) {
    if (!value || !positiveID(value.id)) return null;
    const video = {
      id: positiveID(value.id), user_id: positiveID(value.user_id),
      title: String(value.title || '未命名视频'), description: String(value.description || ''),
      video_url: String(value.video_url || ''), cover_url: String(value.cover_url || ''),
      created_at: String(value.created_at || ''),
    };
    state.videoStore.set(video.id, video);
    return video;
  }

  function authorName(id) {
    return state.user?.id === id ? state.user.username : `创作者 #${id}`;
  }

  function avatar(id, small = false) {
    const node = element('span', `avatar${small ? ' avatar-small' : ''}`, String(id || '?').slice(-2));
    const hues = [245, 194, 24, 154, 279];
    const hue = hues[(Number(id) || 0) % hues.length];
    node.style.backgroundColor = `hsl(${hue} 55% 94%)`;
    node.style.color = `hsl(${hue} 30% 48%)`;
    node.setAttribute('aria-hidden', 'true');
    return node;
  }

  function fullDate(value) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '发布时间未知';
    return date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
  }

  function shortDate(value) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '新作品';
    const minutes = Math.floor((Date.now() - date.getTime()) / 60000);
    if (minutes < 1) return '刚刚发布';
    if (minutes < 60) return `${minutes} 分钟前`;
    if (minutes < 1440) return `${Math.floor(minutes / 60)} 小时前`;
    if (minutes < 10080) return `${Math.floor(minutes / 1440)} 天前`;
    return date.toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' });
  }

  // 只用固定 SVG 图形绘制缺省封面，用户提供的标题、评论一律通过 textContent 写入。
  function poster(video) {
    const node = element('div', 'poster');
    const art = element('div', 'poster-art');
    const palettes = [
      ['#b9d5e5', '#f8dba5', '#769cab', '#547786', '#3a5e73'],
      ['#dbbfad', '#f6e3c3', '#bf967e', '#9d776b', '#735e67'],
      ['#c4c0e4', '#f5d6ca', '#a39cc9', '#7d78ab', '#5e648d'],
      ['#b9d7cc', '#f3dfb7', '#91b6aa', '#6e988f', '#527c7a'],
    ];
    const [sky, sun, back, mid, front] = palettes[video.id % palettes.length];
    art.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 450" preserveAspectRatio="xMidYMid slice"><rect width="800" height="450" fill="${sky}"/><circle cx="600" cy="125" r="68" fill="${sun}"/><path d="M0 295Q150 120 350 275T800 200V450H0Z" fill="${back}"/><path d="M0 390Q230 210 440 335T800 285V450H0Z" fill="${mid}"/><path d="M0 430Q300 340 500 410T800 350V450H0Z" fill="${front}"/><path d="M390 450Q530 360 450 335Q350 300 438 264" stroke="${sky}" fill="none" stroke-width="13" opacity=".45"/></svg>`;
    Object.assign(art.style, { position: 'absolute', inset: '0' });
    const artSVG = art.firstElementChild;
    Object.assign(artSVG.style, { width: '100%', height: '100%', objectFit: 'cover' });
    const badge = element('span', 'poster-label', '暂无封面');
    node.append(art, badge);
    const cover = safeURL(video.cover_url);
    if (cover) {
      const image = new Image();
      image.alt = '';
      image.loading = 'lazy';
      image.decoding = 'async';
      image.referrerPolicy = 'no-referrer';
      image.hidden = true;
      image.addEventListener('load', () => { image.hidden = false; art.hidden = true; badge.hidden = true; }, { once: true });
      image.addEventListener('error', () => image.remove(), { once: true });
      image.src = cover;
      node.append(image);
    }
    return node;
  }

  function toast(message, error = false) {
    const node = element('div', `toast${error ? ' error' : ''}`);
    node.append(icon(error ? 'alert' : 'check'), element('span', '', message));
    const close = button('', '', 'icon-button', 'close');
    close.setAttribute('aria-label', '关闭提示');
    close.addEventListener('click', () => node.remove());
    node.append(close);
    const host = $('auth-dialog').open ? $('auth-dialog') : $('publish-dialog').open ? $('publish-dialog') : $('watch-dialog').open ? $('watch-dialog') : document.body;
    host.append($('toasts'));
    $('toasts').append(node);
    while ($('toasts').children.length > 3) $('toasts').firstElementChild.remove();
    setTimeout(() => node.remove(), error ? 6500 : 4200);
  }

  function formError(id, message = '') {
    $(id).textContent = message;
    $(id).hidden = !message;
  }

  function readSession() {
    try {
      const value = JSON.parse(sessionStorage.getItem(SESSION_KEY));
      if (typeof value?.token === 'string' && value.token && positiveID(value.user?.id) && typeof value.user?.username === 'string') {
        state.token = value.token;
        state.user = { id: positiveID(value.user.id), username: value.user.username.slice(0, 50) };
      }
    } catch { /* 浏览器禁用会话存储时，登录仍能在当前页面使用。 */ }
  }

  function setSession(value) {
    state.token = value?.token || '';
    state.user = value?.user || null;
    try {
      if (value) sessionStorage.setItem(SESSION_KEY, JSON.stringify(value));
      else sessionStorage.removeItem(SESSION_KEY);
    } catch { /* 不阻断页面操作。 */ }
    state.liked = null;
    state.following = null;
    renderAccount();
    if (state.currentVideo) {
      updateWatchIdentity();
      renderComments();
      void loadInteraction();
    }
    if (state.feed === 'following') void loadFeed(true);
    else renderFeed();
  }

  async function request(path, options = {}) {
    const { method = 'GET', body, auth = false, signal } = options;
    const token = state.token;
    if (auth && !token) throw new ApiError('请先登录', 401);
    const controller = new AbortController();
    let timedOut = false;
    const cancel = () => controller.abort();
    if (signal?.aborted) cancel();
    else signal?.addEventListener('abort', cancel, { once: true });
    const timer = setTimeout(() => { timedOut = true; controller.abort(); }, 15000);
    try {
      const headers = { Accept: 'application/json' };
      if (auth) headers.Authorization = `Bearer ${token}`;
      if (body !== undefined) headers['Content-Type'] = 'application/json; charset=utf-8';
      const response = await fetch(`/api/v1${path}`, {
        method, headers, body: body === undefined ? undefined : JSON.stringify(body),
        signal: controller.signal, credentials: 'same-origin', cache: 'no-store',
      });
      const text = await response.text();
      let data = null;
      try { data = text ? JSON.parse(text) : null; } catch { /* 非 JSON 错误由状态码处理。 */ }
      if (!response.ok) {
        if (response.status === 401 && auth && state.token === token) {
          setSession(null);
          toast('登录已失效，请重新登录', true);
        }
        const fallback = response.status === 503 ? '服务暂时无法处理请求，请稍后重试' : `请求失败（${response.status}）`;
        throw new ApiError(data?.error || fallback, response.status);
      }
      return data;
    } catch (error) {
      if (error instanceof ApiError) throw error;
      if (signal?.aborted) throw new DOMException('请求已取消', 'AbortError');
      throw new ApiError(timedOut ? '请求超时，请稍后重试' : '连接失败，请确认服务正在运行');
    } finally {
      clearTimeout(timer);
      signal?.removeEventListener('abort', cancel);
    }
  }

  function renderAccount() {
    const account = $('account');
    account.replaceChildren();
    if (!state.user) {
      account.append(button('登录 / 注册', 'login', 'button button-outline'));
      return;
    }
    account.append(avatar(state.user.id), element('span', 'account-name', state.user.username));
    const logout = button('', 'logout', 'icon-button', 'out');
    logout.title = '退出登录';
    logout.setAttribute('aria-label', '退出登录');
    logout.disabled = state.loggingOut;
    account.append(logout);
  }

  function updateNavigation() {
    document.querySelectorAll('[data-feed]').forEach((node) => {
      const active = node.dataset.feed === state.feed;
      node.classList.toggle('active', active);
      if (active) node.setAttribute('aria-current', 'page');
      else node.removeAttribute('aria-current');
    });
    const info = FEEDS[state.feed];
    $('feed-title').replaceChildren(document.createTextNode(info.title), element('span', 'title-dot', '。'));
    $('feed-description').textContent = info.description;
    document.title = `${state.feed === 'latest' ? '发现视频' : info.tag} · 映流`;
  }

  function switchFeed(feed) {
    if (!FEEDS[feed]) return;
    state.feed = feed;
    state.search = '';
    $('search-input').value = '';
    state.hotLimit = 20;
    updateNavigation();
    void loadFeed(true);
  }

  async function loadFeed(reset = true) {
    if (state.loading && !reset) return;
    state.feedController?.abort();
    const version = ++state.feedVersion;
    const controller = new AbortController();
    state.feedController = controller;
    if (reset) {
      state.videos = []; state.page = 0; state.cursor = ''; state.hasMore = false;
      state.total = 0; state.degraded = false;
    }
    state.feedError = '';
    if (state.feed === 'following' && !state.user) {
      state.loading = false;
      renderFeed();
      return;
    }
    state.loading = true;
    renderFeed();
    const feed = state.feed;
    const page = state.page + 1;
    let path;
    if (feed === 'latest') path = `/videos?page=${page}&page_size=${PAGE_SIZE}`;
    else if (feed === 'following') path = `/feed/following?page_size=${PAGE_SIZE}${state.cursor ? `&cursor=${encodeURIComponent(state.cursor)}` : ''}`;
    else path = `/feed/hot?limit=${state.hotLimit}`;
    try {
      const data = await request(path, { auth: feed === 'following', signal: controller.signal });
      if (version !== state.feedVersion) return;
      const values = feed === 'latest' ? data?.items : data?.videos;
      const incoming = (Array.isArray(values) ? values : []).map(normalizeVideo).filter(Boolean);
      const all = reset || feed === 'hot' ? incoming : [...state.videos, ...incoming];
      state.videos = [...new Map(all.map((video) => [video.id, video])).values()];
      state.page = page;
      state.cursor = typeof data?.next_cursor === 'string' ? data.next_cursor : '';
      state.total = Number(data?.total) || 0;
      state.degraded = Boolean(data?.degraded);
      if (feed === 'latest') state.hasMore = page * PAGE_SIZE < state.total && incoming.length > 0;
      else if (feed === 'following') state.hasMore = Boolean(data?.has_more) && Boolean(state.cursor);
      else state.hasMore = incoming.length >= state.hotLimit && state.hotLimit < 50;
    } catch (error) {
      if (version === state.feedVersion && error.name !== 'AbortError') state.feedError = error.message;
    } finally {
      if (version === state.feedVersion) { state.loading = false; renderFeed(); }
    }
  }

  function renderSpotlight(video) {
    const node = button('', 'open-video', 'spotlight-card');
    node.dataset.videoId = video.id;
    node.setAttribute('aria-label', `播放：${video.title}`);
    node.replaceChildren(poster(video));
    const caption = element('div', 'spotlight-caption');
    const tag = state.feed === 'hot' && state.degraded ? '最新发布' : FEEDS[state.feed].tag;
    caption.append(element('span', 'spotlight-tag', tag), element('h2', '', video.title), element('p', '', `${authorName(video.user_id)}　${shortDate(video.created_at)}`));
    const play = element('span', 'spotlight-play');
    play.append(icon('play'));
    node.append(caption, play);
    return node;
  }

  function renderCard(video) {
    const card = element('article', 'video-card');
    const cover = button('', 'open-video', 'video-cover');
    cover.dataset.videoId = video.id;
    cover.setAttribute('aria-label', `播放：${video.title}`);
    cover.replaceChildren(poster(video));
    const play = element('span', 'cover-play');
    play.append(icon('play'));
    cover.append(play, element('span', 'cover-time', shortDate(video.created_at)));
    const title = button(video.title, 'open-video', 'video-title');
    title.dataset.videoId = video.id;
    title.title = video.title;
    const meta = element('div', 'video-meta');
    meta.append(avatar(video.user_id, true), element('span', 'author-name', authorName(video.user_id)));
    card.append(cover, title, meta);
    return card;
  }

  function showEmpty(title, description, action, actionText, iconName = 'film') {
    const node = $('feed-empty');
    node.replaceChildren(icon(iconName), element('h2', '', title), element('p', '', description), button(actionText, action));
    node.hidden = false;
  }

  function renderFeed() {
    const feature = $('spotlight');
    const grid = $('video-grid');
    feature.replaceChildren(); grid.replaceChildren();
    $('feed-empty').hidden = true;
    $('grid-heading').hidden = true;
    $('load-more').hidden = true;
    $('end-note').hidden = true;
    $('feed-banner').hidden = true;
    $('refresh-feed').disabled = state.loading;
    grid.setAttribute('aria-busy', String(state.loading));
    if (state.feed === 'following' && !state.user) {
      showEmpty('登录后，接着看你喜欢的人', '关注感兴趣的创作者，他们发布的视频会出现在这里。', 'login', '登录 / 注册', 'users');
      return;
    }
    if (state.loading && !state.videos.length) {
      const loading = element('div', 'skeleton skeleton-spotlight');
      loading.setAttribute('aria-label', '正在加载视频');
      feature.append(loading);
      for (let i = 0; i < 6; i++) grid.append(element('div', 'skeleton skeleton-card'));
      $('grid-heading').hidden = false;
      $('grid-title').textContent = '正在加载';
      $('result-count').textContent = '';
      return;
    }
    if (state.feedError && !state.videos.length) {
      showEmpty('内容暂时没有加载出来', state.feedError, 'refresh', '重新加载', 'refresh');
      return;
    }
    const query = state.search.trim().toLocaleLowerCase();
    const videos = state.videos.filter((video) => !query || `${video.title} ${video.description} ${authorName(video.user_id)}`.toLocaleLowerCase().includes(query));
    if (state.feedError || state.degraded) {
      $('feed-banner').textContent = state.feedError || '热门内容暂时不可用，先看看最新发布的作品。';
      $('feed-banner').hidden = false;
    }
    if (!videos.length) {
      if (query) showEmpty('这些作品里还没有找到', '搜索仅覆盖已经加载的视频，可以换个关键词，或继续加载更多作品。', 'clear-search', '清除搜索', 'search');
      else if (state.feed === 'following') showEmpty('这里还在等你的第一份关注', '去发现页打开一个视频，关注喜欢的创作者，再回来看看。', 'discover', '去发现视频', 'users');
      else if (state.feed === 'hot') showEmpty('热榜正在等待第一份喜欢', '去发现页，为喜欢的视频点赞或发表评论。', 'discover', '去发现视频', 'fire');
      else showEmpty('第一段故事，由你开始', '准备一条可播放的视频链接，分享你的第一个作品。', 'publish', '发布视频');
    } else {
      const showFeature = !query;
      if (showFeature) feature.append(renderSpotlight(videos[0]));
      const cards = showFeature ? videos.slice(1) : videos;
      $('grid-heading').hidden = false;
      $('grid-title').textContent = query ? '搜索结果' : cards.length ? '更多作品' : '当前作品';
      $('result-count').textContent = query ? `找到 ${videos.length} 个视频` : `已加载 ${state.videos.length} 个视频`;
      cards.forEach((video) => grid.append(renderCard(video)));
      $('end-note').hidden = state.hasMore || state.loading;
    }
    $('load-more').hidden = !state.hasMore;
    $('load-more').disabled = state.loading;
    $('load-more').textContent = state.loading ? '正在加载…' : '加载更多';
  }

  async function loadRanking() {
    state.rankingController?.abort();
    const controller = new AbortController();
    state.rankingController = controller;
    const version = ++state.rankingVersion;
    state.rankingLoading = true; state.rankingError = '';
    renderRanking();
    try {
      const data = await request('/feed/hot?limit=5', { signal: controller.signal });
      if (version !== state.rankingVersion) return;
      state.ranking = (Array.isArray(data?.videos) ? data.videos : []).map(normalizeVideo).filter(Boolean);
      state.rankingDegraded = Boolean(data?.degraded);
    } catch (error) {
      if (version === state.rankingVersion && error.name !== 'AbortError') state.rankingError = error.message;
    } finally {
      if (version === state.rankingVersion) { state.rankingLoading = false; renderRanking(); }
    }
  }

  function renderRanking() {
    $('ranking-title').replaceChildren(icon(state.rankingDegraded ? 'film' : 'fire'), document.createTextNode(state.rankingDegraded ? '最新发布' : '24 小时热榜'));
    $('ranking-description').textContent = state.rankingDegraded ? '热榜暂时不可用，先看看新作品' : '正在被喜欢的内容';
    const list = $('ranking-list');
    list.replaceChildren();
    if (state.rankingLoading && !state.ranking.length) { list.append(element('li', 'rail-message', '正在寻找好内容…')); return; }
    if (state.rankingError) { list.append(element('li', 'rail-message', '榜单暂时无法加载，可以点击下方重试。')); return; }
    if (!state.ranking.length) { list.append(element('li', 'rail-message', '还没有上榜作品。为喜欢的视频点赞，让它被更多人看见。')); return; }
    state.ranking.forEach((video, index) => {
      const li = element('li');
      const item = button('', 'open-video', 'rank-item');
      item.dataset.videoId = video.id;
      item.replaceChildren(element('span', 'rank-number', String(index + 1)));
      const cover = element('span', 'rank-cover'); cover.append(poster(video));
      const text = element('span', 'rank-text');
      text.append(element('strong', '', video.title), element('small', '', authorName(video.user_id)));
      item.append(cover, text); li.append(item); list.append(li);
    });
  }

  function scheduleRankingRefresh() {
    clearTimeout(state.refreshTimer);
    // 热度异步处理，列表还有短期缓存；延后一次更新，避免连点时重复查询。
    state.refreshTimer = setTimeout(() => void loadRanking(), 6500);
  }

  function openAuth(mode = 'login', afterLogin = '') {
    if (state.authPending) return;
    state.afterLogin = afterLogin;
    setAuthMode(mode);
    if (!$('auth-dialog').open) $('auth-dialog').showModal();
    $('auth-username').focus();
  }

  function setAuthMode(mode) {
    state.authMode = mode === 'register' ? 'register' : 'login';
    const register = state.authMode === 'register';
    $('login-tab').classList.toggle('selected', !register);
    $('register-tab').classList.toggle('selected', register);
    $('login-tab').setAttribute('aria-pressed', String(!register));
    $('register-tab').setAttribute('aria-pressed', String(register));
    $('confirm-password-field').hidden = !register;
    $('auth-confirm').required = register;
    $('auth-password').autocomplete = register ? 'new-password' : 'current-password';
    $('auth-submit').textContent = register ? '注册并登录' : '登录';
    $('auth-title').textContent = register ? '从这里，开启分享' : '欢迎来到映流';
    formError('auth-error');
  }

  async function submitAuth(event) {
    event.preventDefault();
    if (state.authPending) return;
    const username = $('auth-username').value.trim();
    const password = $('auth-password').value;
    const register = state.authMode === 'register';
    if ([...username].length < 3) { formError('auth-error', '用户名至少需要 3 个字符'); return; }
    if ([...password].length < 8) { formError('auth-error', '密码至少需要 8 个字符'); return; }
    if (new TextEncoder().encode(password).length > 72) { formError('auth-error', '密码的 UTF-8 长度不能超过 72 字节，请缩短密码'); return; }
    if (register && password !== $('auth-confirm').value) { formError('auth-error', '两次输入的密码不一致'); return; }
    state.authPending = true;
    $('auth-submit').disabled = true; $('login-tab').disabled = true; $('register-tab').disabled = true;
    $('auth-submit').textContent = register ? '正在创建账号…' : '正在登录…';
    formError('auth-error');
    let created = false;
    try {
      const body = { username, password };
      if (register) { await request('/users', { method: 'POST', body }); created = true; }
      const data = await request('/login', { method: 'POST', body });
      if (!positiveID(data?.id) || typeof data?.token !== 'string' || !data.token) throw new ApiError('登录响应不完整，请稍后重试');
      const next = state.afterLogin;
      setSession({ token: data.token, user: { id: positiveID(data.id), username: String(data.username || username) } });
      $('auth-dialog').close(); $('auth-form').reset();
      toast(register ? '注册成功，已为你登录' : '登录成功，欢迎回来');
      if (next === 'publish') openPublish();
    } catch (error) {
      if (created) setAuthMode('login');
      formError('auth-error', created ? '账号已创建，但自动登录未完成，请切换到登录后重试' : error.message);
    } finally {
      state.authPending = false;
      $('auth-submit').disabled = false; $('login-tab').disabled = false; $('register-tab').disabled = false;
      $('auth-submit').textContent = state.authMode === 'register' ? '注册并登录' : '登录';
    }
  }

  async function logout() {
    if (!state.user || state.loggingOut) return;
    state.loggingOut = true; renderAccount();
    try {
      await request('/logout', { method: 'POST', auth: true });
      setSession(null);
      toast('已退出登录');
    } catch (error) { if (error.status !== 401) toast(error.message, true); }
    finally { state.loggingOut = false; renderAccount(); }
  }

  function openPublish() {
    if (!state.user) { openAuth('login', 'publish'); return; }
    formError('publish-error');
    if (!$('publish-dialog').open) $('publish-dialog').showModal();
    $('publish-name').focus();
  }

  async function submitPublish(event) {
    event.preventDefault();
    if (state.publishing) return;
    if (!state.user) { openAuth('login', 'publish'); return; }
    const body = {
      title: $('publish-name').value.trim(), description: $('publish-description').value.trim(),
      video_url: $('publish-url').value.trim(), cover_url: $('publish-cover').value.trim(),
    };
    if (!body.title) { formError('publish-error', '请填写视频标题'); return; }
    if (!safeURL(body.video_url) || (body.cover_url && !safeURL(body.cover_url))) {
      formError('publish-error', '视频和封面地址需要以 http:// 或 https:// 开头，且不包含账号密码'); return;
    }
    const token = state.token;
    state.publishing = true; $('publish-submit').disabled = true;
    $('publish-submit').textContent = '正在发布…'; formError('publish-error');
    try {
      const data = await request('/videos', { method: 'POST', body, auth: true });
      if (state.token !== token) return;
      normalizeVideo(data);
      $('publish-dialog').close(); $('publish-form').reset();
      toast('视频发布成功'); switchFeed('latest');
    } catch (error) { formError('publish-error', error.message); }
    finally { state.publishing = false; $('publish-submit').disabled = false; $('publish-submit').textContent = '发布视频'; }
  }

  function watchCurrent(version, token) {
    return $('watch-dialog').open && version === state.watchVersion && (token === undefined || token === state.token);
  }

  async function openVideo(id) {
    const videoID = positiveID(id);
    if (!videoID) return;
    const intent = ++state.openVersion;
    try {
      const video = state.videoStore.get(videoID) || normalizeVideo(await request(`/videos/${videoID}`));
      if (intent !== state.openVersion) return;
      if (!video) throw new ApiError('视频不存在', 404);
      state.commentsController?.abort();
      state.watchVersion++;
      state.currentVideo = video;
      state.liked = null; state.following = null; state.likeBusy = false; state.followBusy = false;
      state.comments = []; state.commentsError = ''; state.deleting.clear();
      $('comment-content').value = ''; formError('comment-error');
      $('watch-title').textContent = video.title;
      $('watch-description').textContent = video.description || '这位创作者还没有填写视频简介。';
      $('watch-date').textContent = fullDate(video.created_at);
      updateWatchIdentity();
      const player = $('watch-video');
      player.pause(); player.removeAttribute('src'); player.removeAttribute('poster');
      const cover = safeURL(video.cover_url);
      if (cover) player.poster = cover;
      const source = safeURL(video.video_url);
      $('original-video').hidden = !source;
      if (source) $('original-video').href = source;
      else $('original-video').removeAttribute('href');
      $('playback-message').hidden = true;
      if (!$('watch-dialog').open) $('watch-dialog').showModal();
      if (!source || isExampleURL(source)) {
        player.load();
        showPlaybackError(isExampleURL(source) ? '当前保存的是示例链接，还没有可播放的视频。请发布一个真实视频地址；评论、点赞和关注仍然可以使用。' : '视频地址无效。请使用可直接播放的 HTTP 或 HTTPS 视频链接。');
      } else { player.src = source; player.load(); }
      const url = new URL(location.href); url.searchParams.set('video', String(videoID));
      history.replaceState(null, '', url);
      void loadComments(); void loadInteraction();
    } catch (error) { toast(error.message, true); }
  }

  function showPlaybackError(message) {
    if (!$('watch-dialog').open) return;
    $('playback-message').hidden = false;
    $('playback-message').querySelector('p').textContent = message || '请检查视频链接是否可访问，以及浏览器是否支持该格式。你仍然可以查看评论、点赞和关注创作者。';
  }

  function updateWatchIdentity() {
    const video = state.currentVideo;
    if (!video) return;
    $('watch-author').textContent = authorName(video.user_id);
    const face = avatar(video.user_id);
    $('watch-avatar').replaceWith(face); face.id = 'watch-avatar';
    const own = state.user?.id === video.user_id;
    const follow = $('follow-button');
    follow.disabled = own || state.followBusy || Boolean(state.user && (state.statusLoading || state.following === null));
    follow.textContent = own ? '我的作品' : state.followBusy ? '处理中…' : state.following ? '已关注' : '关注';
    follow.classList.toggle('is-following', Boolean(state.following));
    const like = $('like-button');
    like.disabled = state.likeBusy || Boolean(state.user && (state.statusLoading || state.liked === null));
    like.setAttribute('aria-pressed', String(Boolean(state.liked)));
    like.querySelector('span').textContent = state.likeBusy ? '处理中…' : state.liked ? '已点赞' : '点赞';
    const hint = $('interaction-hint'); hint.replaceChildren();
    if (!state.user) hint.textContent = '登录后参与互动';
    else if (state.statusLoading) hint.textContent = '正在读取互动状态…';
    else if (state.liked === null || (!own && state.following === null)) hint.append(button('重试互动状态', 'retry-state', 'text-button'));
    $('comment-content').disabled = !state.user;
    $('comment-content').placeholder = state.user ? '聊聊你的想法…' : '登录后，留下你的想法';
    $('comment-login').hidden = Boolean(state.user);
    $('comment-submit').hidden = !state.user;
    $('comment-limit').hidden = !state.user;
    updateCommentButton();
  }

  async function loadInteraction() {
    const video = state.currentVideo;
    if (!video || !state.user || !$('watch-dialog').open) { state.statusLoading = false; updateWatchIdentity(); return; }
    const version = state.watchVersion; const token = state.token;
    state.statusLoading = true; updateWatchIdentity();
    const results = await Promise.allSettled([
      request(`/videos/${video.id}/like`, { auth: true }),
      state.user.id === video.user_id ? Promise.resolve({ following: false }) : request(`/users/${video.user_id}/follow`, { auth: true }),
    ]);
    if (!watchCurrent(version, token)) return;
    state.liked = results[0].status === 'fulfilled' ? Boolean(results[0].value?.liked) : null;
    state.following = results[1].status === 'fulfilled' ? Boolean(results[1].value?.following) : null;
    state.statusLoading = false; updateWatchIdentity();
  }

  async function changeLike() {
    if (!state.user) { openAuth(); return; }
    const video = state.currentVideo;
    if (!video || state.likeBusy || state.liked === null || state.statusLoading) return;
    const desired = !state.liked; const version = state.watchVersion; const token = state.token;
    state.likeBusy = true; updateWatchIdentity();
    try {
      await request(`/videos/${video.id}/like`, { method: desired ? 'POST' : 'DELETE', auth: true });
      if (!watchCurrent(version, token)) return;
      state.liked = desired; toast(desired ? '已点赞' : '已取消点赞'); scheduleRankingRefresh();
    } catch (error) {
      if (watchCurrent(version, token)) {
        if (error.status === 409 && desired) { state.liked = true; toast('你已经点赞过这个视频'); }
        else toast(error.message, true);
      }
    } finally { if (watchCurrent(version, token)) { state.likeBusy = false; updateWatchIdentity(); } }
  }

  async function changeFollow() {
    if (!state.user) { openAuth(); return; }
    const video = state.currentVideo;
    if (!video || state.followBusy || state.following === null || state.statusLoading || video.user_id === state.user.id) return;
    const desired = !state.following; const version = state.watchVersion; const token = state.token;
    state.followBusy = true; updateWatchIdentity();
    try {
      await request(`/users/${video.user_id}/follow`, { method: desired ? 'POST' : 'DELETE', auth: true });
      if (!watchCurrent(version, token)) return;
      state.following = desired; toast(desired ? '已关注这位创作者' : '已取消关注');
      if (state.feed === 'following') void loadFeed(true);
    } catch (error) {
      if (watchCurrent(version, token)) {
        if (error.status === 409 && desired) { state.following = true; toast('你已经关注了这位创作者'); }
        else if (error.status === 404 && !desired) { state.following = false; toast('已取消关注'); }
        else toast(error.message, true);
      }
    } finally { if (watchCurrent(version, token)) { state.followBusy = false; updateWatchIdentity(); } }
  }

  async function loadComments() {
    const video = state.currentVideo;
    if (!video || !$('watch-dialog').open) return;
    state.commentsController?.abort();
    const controller = new AbortController(); state.commentsController = controller;
    const version = state.watchVersion;
    state.commentsLoading = true; state.commentsError = ''; renderComments(); updateCommentButton();
    try {
      const data = await request(`/videos/${video.id}/comments`, { signal: controller.signal });
      if (!watchCurrent(version) || controller.signal.aborted) return;
      state.comments = (Array.isArray(data) ? data : []).filter((item) => positiveID(item.id)).reverse();
    } catch (error) {
      if (watchCurrent(version) && !controller.signal.aborted && error.name !== 'AbortError') state.commentsError = error.message;
    } finally {
      if (watchCurrent(version) && state.commentsController === controller) {
        state.commentsLoading = false; renderComments(); updateCommentButton();
      }
    }
  }

  function renderComments() {
    const list = $('comments-list'); list.replaceChildren();
    list.setAttribute('aria-busy', String(state.commentsLoading));
    $('comment-count').textContent = state.commentsLoading ? '' : String(state.comments.length);
    if (state.commentsLoading && !state.comments.length) { list.append(element('p', 'comments-empty', '正在加载评论…')); return; }
    if (state.commentsError) {
      const message = element('div', 'comments-empty', state.commentsError);
      message.append(element('br'), button('重新加载', 'retry-comments', 'text-button'));
      list.append(message); return;
    }
    if (!state.comments.length) { list.append(element('p', 'comments-empty', '还没有评论，来聊聊你的想法。')); return; }
    state.comments.forEach((comment) => {
      const item = element('article', 'comment-item');
      const userID = positiveID(comment.user_id);
      item.append(avatar(userID, true));
      const body = element('div', 'comment-body');
      const meta = element('div', 'comment-meta');
      meta.append(element('strong', '', authorName(userID)));
      if (state.user?.id === userID) {
        const remove = button(state.deleting.has(comment.id) ? '删除中…' : '删除', 'delete-comment', 'text-button comment-delete');
        remove.dataset.commentId = comment.id; remove.disabled = state.deleting.has(comment.id);
        remove.setAttribute('aria-label', '删除这条评论'); meta.append(remove);
      }
      body.append(meta, element('p', '', String(comment.content || '')), element('time', '', fullDate(comment.created_at)));
      item.append(body); list.append(item);
    });
  }

  function updateCommentButton() {
    $('comment-limit').textContent = `${$('comment-content').value.length} / 2000`;
    $('comment-submit').disabled = !state.user || state.commentBusy || state.commentsLoading || !$('comment-content').value.trim();
    $('comment-submit').textContent = state.commentBusy ? '发送中…' : '发送';
  }

  async function submitComment(event) {
    event.preventDefault();
    if (!state.user) { openAuth(); return; }
    if (state.commentBusy || !state.currentVideo) return;
    const content = $('comment-content').value.trim();
    if (!content) return;
    const id = state.currentVideo.id; const version = state.watchVersion; const token = state.token;
    state.commentBusy = true; updateCommentButton(); formError('comment-error');
    try {
      await request(`/videos/${id}/comments`, { method: 'POST', body: { content }, auth: true });
      if (!watchCurrent(version, token)) return;
      $('comment-content').value = ''; toast('评论已发布'); scheduleRankingRefresh();
      await loadComments();
    } catch (error) { if (watchCurrent(version, token)) formError('comment-error', error.message); }
    finally { state.commentBusy = false; updateCommentButton(); }
  }

  async function deleteComment(id) {
    const commentID = positiveID(id);
    if (!state.user || !commentID || state.deleting.has(commentID)) return;
    if (!window.confirm('删除这条评论？')) return;
    const version = state.watchVersion; const token = state.token;
    state.deleting.add(commentID); renderComments();
    try {
      await request(`/comments/${commentID}`, { method: 'DELETE', auth: true });
      if (!watchCurrent(version, token)) return;
      state.comments = state.comments.filter((comment) => comment.id !== commentID);
      toast('评论已删除'); scheduleRankingRefresh();
    } catch (error) { if (watchCurrent(version, token)) toast(error.message, true); }
    finally { state.deleting.delete(commentID); if (watchCurrent(version)) renderComments(); }
  }

  async function shareVideo() {
    if (!state.currentVideo) return;
    const url = new URL('/', location.origin);
    url.searchParams.set('video', String(state.currentVideo.id));
    try {
      if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable');
      await navigator.clipboard.writeText(url.href); toast('视频链接已复制');
    } catch { window.prompt('复制这个视频链接', url.href); }
  }

  // 统一事件入口；所有带业务含义的按钮都连接到真实 API 或明确的页面操作。
  document.addEventListener('click', (event) => {
    const target = event.target.closest?.('button');
    if (!target || target.disabled) return;
    if (target.dataset.feed) { switchFeed(target.dataset.feed); return; }
    if (target.dataset.close) { $(target.dataset.close)?.close(); return; }
    if (target.dataset.authMode) { setAuthMode(target.dataset.authMode); return; }
    switch (target.dataset.action) {
      case 'login': openAuth(); break;
      case 'logout': void logout(); break;
      case 'publish': openPublish(); break;
      case 'discover': switchFeed('latest'); break;
      case 'refresh': void loadFeed(true); void loadRanking(); break;
      case 'refresh-ranking': void loadRanking(); break;
      case 'load-more':
        if (state.feed === 'hot') state.hotLimit = Math.min(50, state.hotLimit + 10);
        void loadFeed(false); break;
      case 'clear-search': state.search = ''; $('search-input').value = ''; renderFeed(); break;
      case 'open-video': void openVideo(target.dataset.videoId); break;
      case 'like': void changeLike(); break;
      case 'follow': void changeFollow(); break;
      case 'share': void shareVideo(); break;
      case 'retry-state': void loadInteraction(); break;
      case 'retry-comments': void loadComments(); break;
      case 'delete-comment': void deleteComment(target.dataset.commentId); break;
    }
  });

  $('search-form').addEventListener('submit', (event) => event.preventDefault());
  $('search-input').addEventListener('input', () => { state.search = $('search-input').value; renderFeed(); });
  $('auth-form').addEventListener('submit', (event) => void submitAuth(event));
  $('publish-form').addEventListener('submit', (event) => void submitPublish(event));
  $('comment-form').addEventListener('submit', (event) => void submitComment(event));
  $('comment-content').addEventListener('input', () => { updateCommentButton(); formError('comment-error'); });
  $('watch-video').addEventListener('error', () => showPlaybackError());
  $('auth-dialog').addEventListener('close', () => { state.afterLogin = ''; });
  $('watch-dialog').addEventListener('close', () => {
    state.openVersion++; state.watchVersion++; state.currentVideo = null; state.commentsController?.abort();
    const player = $('watch-video'); player.pause(); player.removeAttribute('src'); player.removeAttribute('poster'); player.load();
    const url = new URL(location.href); url.searchParams.delete('video'); history.replaceState(null, '', url);
  });
  document.querySelectorAll('dialog').forEach((dialog) => {
    dialog.addEventListener('close', () => { if (dialog.contains($('toasts'))) document.body.append($('toasts')); });
    dialog.addEventListener('click', (event) => {
      if (event.target !== dialog) return;
      const bounds = dialog.getBoundingClientRect();
      if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) dialog.close();
    });
  });

  async function boot() {
    readSession(); renderAccount(); updateNavigation();
    state.loading = true; renderFeed();
    if (state.token) {
      try {
        const me = await request('/me', { auth: true });
        if (positiveID(me?.user_id) !== state.user?.id) setSession(null);
      } catch (error) { if (error.status !== 401) toast('登录状态暂时无法确认，你可以继续浏览公开内容', true); }
    }
    void loadFeed(true); void loadRanking();
    const linkedVideo = positiveID(new URL(location.href).searchParams.get('video'));
    if (linkedVideo) await openVideo(linkedVideo);
  }

  void boot();
})();
