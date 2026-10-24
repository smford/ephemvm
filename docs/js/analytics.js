/**
 * Google Analytics 4 (GA4) Integration for ephemvm documentation
 * Measurement ID: ephemvm
 */

(function () {
  'use strict';

  var GA_MEASUREMENT_ID = 'ephemvm';

  function getPWAMode() {
    if (window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone === true) {
      return 'standalone';
    }
    return 'browser';
  }

  function trackEvent(eventName, params) {
    try {
      if (typeof window.gtag === 'function') {
        var enrichedParams = Object.assign(
          {
            pwa_mode: getPWAMode(),
            page_location: window.location.href,
            page_path: window.location.pathname + window.location.hash,
            timestamp: new Date().toISOString(),
          },
          params || {}
        );
        window.gtag('event', eventName, enrichedParams);
      }
    } catch (e) {
      // Swallowed to guarantee non-blocking execution
    }
  }

  function trackPageView(path) {
    trackEvent('page_view', {
      page_path: path || window.location.pathname + window.location.hash,
      page_title: document.title,
    });
  }

  function trackCopyCommand(commandText) {
    trackEvent('copy_code_snippet', {
      command: commandText ? commandText.substring(0, 100) : '',
    });
  }

  window.addEventListener('hashchange', function () {
    trackPageView(window.location.hash || '#index.md');
  });

  window.addEventListener('DOMContentLoaded', function () {
    trackPageView(window.location.hash || '#index.md');

    document.addEventListener('click', function (e) {
      var copyBtn = e.target.closest('.copy-code-btn, .copy-button, pre button');
      if (copyBtn) {
        var pre = copyBtn.closest('pre');
        if (pre) {
          trackCopyCommand(pre.innerText);
        }
      }
    });
  });

  window.ephemvmAnalytics = {
    trackEvent: trackEvent,
    trackPageView: trackPageView,
    trackCopyCommand: trackCopyCommand,
  };
})();
