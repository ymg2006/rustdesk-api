'use strict';
const CACHE_VERSION = '20260801043000';
const MANIFEST = 'flutter-app-manifest-' + CACHE_VERSION;
const TEMP = 'flutter-temp-cache-' + CACHE_VERSION;
const CACHE_NAME = 'flutter-app-cache-' + CACHE_VERSION;

const RESOURCES = {"/":"6c88330f5cdc6b102e286bdd5f1283f2","assets/AssetManifest.bin":"917644686808dc844d51a83e4d00a519","assets/AssetManifest.bin.json":"a7e320ec42d2cd3ce17f1b55689251b8","assets/AssetManifest.json":"5b9b4b29531b22bb35af66f6a5bbea7b","assets/FontManifest.json":"f4f31d18ca443c7bfb50340e84b860d5","assets/NOTICES":"4097c7a02b4cbfc878b09cd3223f65b8","assets/assets/actions.svg":"2a43efdc71e95b7db43c1fc5ae34ed87","assets/assets/actions_mobile.svg":"840555746fccec5610779500eec942be","assets/assets/address_book.ttf":"612eb0515c3bca0ea7e661cb74c14fcc","assets/assets/android.svg":"2172ec92c0da8b72dfe31565d5cee45c","assets/assets/arrow.svg":"ef5695b014466fc2f60dc3a201fa9435","assets/assets/auth-apple.svg":"f96f69a4e4e7f5415d0888e249198b8a","assets/assets/auth-auth0.svg":"563377c91144cb7d1a02ff8ee5aa58bb","assets/assets/auth-default.svg":"6c1a2702edb372085ca4371c4ef09aa5","assets/assets/auth-facebook.svg":"ee6b89b302080bcef63b9779b6c9bea4","assets/assets/auth-github.svg":"3c30cd99eb7418fe737b717536672201","assets/assets/auth-gitlab.svg":"b9b4d037f50930335a0e503cd9a35294","assets/assets/auth-google.svg":"26dd5d10ffb0d02eb37a3858f2e533f3","assets/assets/auth-microsoft.svg":"22f8f6cc1c8e024fe205d3f56f2be58d","assets/assets/auth-okta.svg":"2f417879cb6ff130f0b6294c6710fa72","assets/assets/call_end.svg":"82190401a712a19798a416252071b2ae","assets/assets/call_wait.svg":"ca0cdeaa2a498f901db4973419f56e49","assets/assets/chat.svg":"10595dcc8484d6f7f140f54d6db32877","assets/assets/chat2.svg":"82fee3bfc120214c67548759d63ce865","assets/assets/checkbox-outline.svg":"f0d7b636853657cc21df676e2f473e1f","assets/assets/chevron_up_chevron_down.svg":"9ca13e04aa9ce89f591d775461e827d2","assets/assets/close.svg":"e2f1a0bd1efe55a14fc21e70d14265e8","assets/assets/device_group.ttf":"0c14da6ee5eab333bfacf34943d68887","assets/assets/display.svg":"bf281b8b99eac956f7ab03028e7c235b","assets/assets/display_switcher.svg":"77a4237b2f6764becf13f8f83b592bd4","assets/assets/dots.svg":"10dc01da59bbf56899fe3dfb24de4588","assets/assets/file.svg":"8e000fe8e15b1ecb0c88d46ee64f6265","assets/assets/file_transfer.svg":"7530de7176b77ed19834c6dd87a692c4","assets/assets/folder.svg":"eeaa976f375cc066c7fb7cc99733cd30","assets/assets/folder_new.svg":"bf774d77fd7614a7d3f26f98f3e5d8d6","assets/assets/fullscreen.svg":"ddb2613848048e0b1ac089eacd72b8b6","assets/assets/fullscreen_exit.svg":"e9adde63a9a4613ce646c2b68a81554a","assets/assets/gestures.ttf":"a70c60208ba07ce378ed9a5cf8aa586b","assets/assets/home.svg":"2320ca704fa8dfa8fadedd4c181b1d34","assets/assets/icon.svg":"9673d0a1dd44d81bc31c76a56857d787","assets/assets/insecure.svg":"58385c3079d8d21b891582151147a144","assets/assets/insecure_relay.svg":"459680c84934bd3e0ed583df62617080","assets/assets/kb_layout_iso.svg":"19463c9f2c8c2ce6db8a57d2bcf206ac","assets/assets/kb_layout_not_iso.svg":"c0903bc16f7a3584677d1129c627f79e","assets/assets/keyboard_mouse.svg":"0c0566939e5d32b85b3fbdd351e6587f","assets/assets/linux.svg":"94446987e0c95123c5549fb26affdd32","assets/assets/mac.svg":"aea75d88e94e3feda3a8e85fe8f7e83b","assets/assets/message_24dp_5F6368.svg":"b0f463bd4acc2aa9ead75c7e652a0543","assets/assets/more.ttf":"d1d2c5484a00438591c383e6ff514747","assets/assets/peer_searchbar.ttf":"f189a84ff155487e21892166a8a1a49b","assets/assets/pinned.svg":"0a2acd10ced63f0224be16aa8c0f3b5f","assets/assets/rec.svg":"8cbc54b941a213eee9017f723787b3c0","assets/assets/record_screen.svg":"ccb57d890c59dad9e4b7d22422ed9b21","assets/assets/refresh.svg":"1d14356f6aec82f5708c1d7bc3538ff4","assets/assets/scam.png":"a6d290cead819b6bfe4248b5d542567f","assets/assets/screen.svg":"c6b0ea973c050844c2c2cba27eaffd23","assets/assets/search.svg":"492f2847532faf43169300c524aa7ccb","assets/assets/secure.svg":"bf1b40b1a434039d84856285af6708ed","assets/assets/secure_relay.svg":"cd3fe0afb93a1978651efc96f723b203","assets/assets/tabbar.ttf":"593f286bbe900c64016ed23dc8ba91d6","assets/assets/transfer.svg":"bde9d26182e2684ae721ecb5a55963fc","assets/assets/trash.svg":"32cb1ca9e5561cc9d545e185d8bf17f1","assets/assets/unpinned.svg":"d529613e4ab67bc9049861fcd8646606","assets/assets/voice_call.svg":"b261a6c0ce34fd6a31bba07f8e32f5fa","assets/assets/voice_call_waiting.svg":"cbbab0ecd11a1c20ca16ebd41e72d86a","assets/assets/win.svg":"e6032ece6ba082b1ab125503577d9525","assets/fonts/MaterialIcons-Regular.otf":"c59a5fead551f063f4a63736a804739b","assets/fonts/fallback/roboto/v20/KFOmCnqEu92Fr1Me5WZLCzYlKw.ttf":"86da78cb59576328483a11c6ef74bc2b","assets/fonts/google/0303f43a87738673f33aa8a225fa6d16bf73204c074598b704a39facf8cbae85.ttf":"913d3f5854a400748e175d8f8bb0b105","assets/fonts/google/043ed62678605a78210b4f88ebbb39ba2f316faade1686e2666e44c1e033cc2c.ttf":"1bfba2cc17983eb5707afd970e077bdc","assets/fonts/google/1a1dc1174e641a40ae663f3dc4b3b1b8580268f2af7766b61c52b41aeaaed062.ttf":"bceaf5cf923c73a5209fad0b57878b54","assets/fonts/google/1eb1c53389bf0ebe5d7c34dfbf1ea0d93bfb861a7d2d038dbf97dc45a5c4f57c.ttf":"04cea0fe45458f3531f661565bc85b9b","assets/fonts/google/82d02ade9ef7730e7c232e04fd708a88b7c6d35f0cf21277f3ad6e38db5573d6.ttf":"57be4db099986a4fb0a315471c6867f8","assets/fonts/google/b1db000c824bd66cb5fc4699fa2e591d8bb897e67dd0a0fb0a6d55ca150737bf.ttf":"61065f8c6b16478e0c7b977d568e7f48","assets/fonts/google/cc6e88fb2890693bb5923ee2b933bb4e35c14c85dd28195e8f8766f682e3abec.ttf":"275b438f7021779d4753c4f6f93b2231","assets/fonts/google/dcfc77cb09baebfeb5200c759d7faaefa0a051c4d99f4f11a6529d7a437b225a.ttf":"df70e68cf0e832eda7d85e9dafb3ff0a","assets/fonts/google/dfe4fef30ea4f90de78a0ce8852e017b9f6107f03c55f732c6cac923cf643df8.ttf":"abd37d33a01da50311c01a77ced668a1","assets/fonts/google/f305ef4e85f3feea6991d194af6b97217ea8eb4ebbda4027bc95cf8de8c17bde.ttf":"7a9a88e8b0343fd66cd7929aeaa3cc7f","assets/packages/dash_chat_2/assets/placeholder.png":"ce1fece6c831b69b75c6c25a60b5b0f3","assets/packages/dash_chat_2/assets/profile_placeholder.png":"77f5794e2eb49f7989b8f85e92cfa4e0","assets/packages/flex_color_picker/assets/opacity.png":"49c4f3bcb1b25364bb4c255edcaaf5b2","assets/packages/wakelock_plus/assets/no_sleep.js":"9c3aa3cd0b217305aa860decab3d9f42","assets/packages/window_manager/images/ic_chrome_close.png":"75f4b8ab3608a05461a31fc18d6b47c2","assets/packages/window_manager/images/ic_chrome_maximize.png":"af7499d7657c8b69d23b85156b60298c","assets/packages/window_manager/images/ic_chrome_minimize.png":"4282cd84cb36edf2efb950ad9269ca62","assets/packages/window_manager/images/ic_chrome_unmaximize.png":"4a90c1909cb74e8f0d35794e2f61d8bf","assets/shaders/ink_sparkle.frag":"9bb2aaa0f9a9213b623947fa682efa76","canvaskit/canvaskit.js":"4a9bf79219d86ed807ac1ea2c30e01dd","canvaskit/canvaskit.js.symbols":"7591a27e90a9f47b73104b5beea5f732","canvaskit/canvaskit.wasm":"1f237a213d7370cf95f443d896176460","canvaskit/chromium/canvaskit.js":"067d1b778b913719f905e9eba6d9f2d4","canvaskit/chromium/canvaskit.js.symbols":"5e3724af47d205af948bfc9946c80dc4","canvaskit/chromium/canvaskit.wasm":"b1ac05b29c127d86df4bcfbf50dd902a","canvaskit/skwasm.js":"9e94c7112288ea6e16844d9879ce08dc","canvaskit/skwasm.js.symbols":"601a3adb24ac6b21b8e89735a27416f3","canvaskit/skwasm.wasm":"9f0c0c02b82a910d12ce0543ec130e60","canvaskit/skwasm.worker.js":"b31cd002f2ed6e6d27aed1fa7658efae","favicon.svg":"c6baa0e25649b906d1dfedef6351252b","ffmpeg-core.js":"6c722ec9e4e15b2ff3f16c2791520e0f","ffmpeg-core.wasm":"c489b260982eb125be0bb8c38f52fd48","ffmpeg.js":"22d9b54cca5554e1c901adf6088fcb4f","flutter.js":"1f53b82fd1bdf0ba28a734c5e1dc13fd","flutter_bootstrap.js":"e22489c358965d4919805253ca3ff338","icons/Icon-192.png":"629929e6a05a91a85fdaaf0dd7ab1068","icons/Icon-512.png":"d861cdc25086ff381baef518a66aaedb","icons/Icon-maskable-192.png":"629929e6a05a91a85fdaaf0dd7ab1068","icons/Icon-maskable-512.png":"d861cdc25086ff381baef518a66aaedb","index.html":"6c88330f5cdc6b102e286bdd5f1283f2","js/dist/index.js":"679d4d171ea110557265ba97dc4e77cd","js/dist/vendor.js":"8ce7a60e7fd51efd8ac1849ed6a136e4","libopus.js":"5b3f168c45519ea785bfbf63c640c161","libopus.wasm":"b8801d4a953d58e739fd9d25134467d3","libs/stream/StreamSaver.min.js":"846005092e6fee79a9bae3514a6122ec","libs/stream/ponyfill.min.js":"e1749398ecf8f956a070bd14a4d71cf8","main.dart.js":"f8a8d5d91a230ae251d966d1a0414220","manifest.json":"2fb7e5d5bb47716811b82c5834d24f10","version.json":"a03aefde192061f29522735eb284a13f"};
const CORE = ["index.html","main.dart.js","flutter_bootstrap.js","js/dist/index.js","js/dist/vendor.js","assets/AssetManifest.bin.json","assets/FontManifest.json","assets/fonts/fallback/roboto/v20/KFOmCnqEu92Fr1Me5WZLCzYlKw.ttf"];

self.addEventListener("install", (event) => {
  self.skipWaiting();
  event.waitUntil(caches.open(TEMP).then((cache) =>
    cache.addAll(CORE.map((value) => new Request(value, {cache: 'reload'})))
  ));
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    try {
      const keep = new Set([MANIFEST, TEMP, CACHE_NAME]);
      for (const cacheName of await caches.keys()) {
        if (cacheName.startsWith('flutter-') && !keep.has(cacheName)) await caches.delete(cacheName);
      }
      let contentCache = await caches.open(CACHE_NAME);
      const tempCache = await caches.open(TEMP);
      const manifestCache = await caches.open(MANIFEST);
      const manifestResponse = await manifestCache.match('manifest');
      if (manifestResponse) {
        const oldManifest = await manifestResponse.json();
        const origin = self.location.origin;
        for (const request of await contentCache.keys()) {
          let key = request.url.substring(origin.length + 1);
          if (key === '') key = '/';
          if (!RESOURCES[key] || RESOURCES[key] !== oldManifest[key]) await contentCache.delete(request);
        }
      } else {
        await caches.delete(CACHE_NAME);
        contentCache = await caches.open(CACHE_NAME);
      }
      for (const request of await tempCache.keys()) {
        const response = await tempCache.match(request);
        if (response) await contentCache.put(request, response);
      }
      await caches.delete(TEMP);
      await manifestCache.put('manifest', new Response(JSON.stringify(RESOURCES)));
      await self.clients.claim();
    } catch (error) {
      console.error('Failed to upgrade service worker: ' + error);
      await caches.delete(CACHE_NAME);
      await caches.delete(TEMP);
      await caches.delete(MANIFEST);
      throw error;
    }
  })());
});

self.addEventListener("fetch", (event) => {
  if (event.request.method !== 'GET') return;
  const origin = self.location.origin;
  let key = event.request.url.substring(origin.length + 1);
  if (key.includes('?v=')) key = key.split('?v=')[0];
  if (event.request.url === origin || event.request.url.startsWith(origin + '/#') || key === '') key = '/';
  if (!RESOURCES[key]) return;
  if (key === '/') { event.respondWith(onlineFirst(event)); return; }
  event.respondWith(caches.open(CACHE_NAME).then(async (cache) => {
    const cached = await cache.match(event.request);
    if (cached) return cached;
    const response = await fetch(event.request);
    if (response && response.ok) await cache.put(event.request, response.clone());
    return response;
  }));
});

self.addEventListener('message', (event) => {
  if (event.data === 'skipWaiting') self.skipWaiting();
  else if (event.data === 'downloadOffline') event.waitUntil(downloadOffline());
});

async function downloadOffline() {
  const origin = self.location.origin;
  const contentCache = await caches.open(CACHE_NAME);
  const currentContent = {};
  for (const request of await contentCache.keys()) {
    let key = request.url.substring(origin.length + 1);
    if (key === '') key = '/';
    currentContent[key] = true;
  }
  const resources = Object.keys(RESOURCES).filter((key) => !currentContent[key]);
  await contentCache.addAll(resources);
}

async function onlineFirst(event) {
  try {
    const response = await fetch(event.request);
    const cache = await caches.open(CACHE_NAME);
    if (response && response.ok) await cache.put(event.request, response.clone());
    return response;
  } catch (error) {
    const cache = await caches.open(CACHE_NAME);
    const response = await cache.match(event.request);
    if (response) return response;
    throw error;
  }
}
