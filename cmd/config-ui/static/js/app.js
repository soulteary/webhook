/**
 * Webhook config generator UI.
 * Expects window.I18N from server-rendered page (config/page.yaml).
 */
(function () {
	'use strict';

	var LANG_STORAGE_KEY = 'webhook-config-ui-lang';

	function getLang() {
		return localStorage.getItem(LANG_STORAGE_KEY) || 'zh';
	}

	function applyLang(lang) {
		var I18N = window.I18N;
		if (!I18N) return;
		lang = I18N[lang] ? lang : 'zh';
		localStorage.setItem(LANG_STORAGE_KEY, lang);
		document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en';
		document.title = I18N[lang].title;
		document.querySelectorAll('[data-i18n]').forEach(function (el) {
			var key = el.getAttribute('data-i18n');
			if (key && I18N[lang][key] !== undefined) el.textContent = I18N[lang][key];
		});
		document.querySelectorAll('.lang-switch a').forEach(function (a) {
			a.classList.toggle('active', a.getAttribute('data-lang') === lang);
		});
	}

	document.querySelectorAll('.lang-switch a').forEach(function (a) {
		a.addEventListener('click', function (e) {
			e.preventDefault();
			applyLang(this.getAttribute('data-lang'));
		});
	});
	applyLang(getLang());

	try {
		var savedId = localStorage.getItem('webhook-config-ui-id');
		var savedBase = localStorage.getItem('webhook-config-ui-base-url');
		if (savedId) {
			var el = document.getElementById('id');
			if (el && !el.value) el.value = savedId;
		}
		if (savedBase) {
			var elBase = document.getElementById('webhook_base_url');
			if (elBase && !elBase.value) elBase.value = savedBase;
		}
	} catch (e) { /* ignore */ }

	var EXAMPLE = {
		id: 'example-hook',
		'execute-command': '/bin/true',
		'command-working-directory': '',
		'response-message': 'OK',
		'webhook_base_url': 'http://localhost:9000',
		'http-methods': 'POST',
		'success-http-response-code': 200,
		'include-command-output-in-response': false,
		'response-headers': '',
		'pass-arguments-to-command': '',
		'pass-environment-to-command': '',
		'trigger-rule': '',
		'incoming-payload-content-type': 'application/json'
	};

	function setExample() {
		var set = function (id, value) {
			var el = document.getElementById(id);
			if (!el) return;
			if (el.type === 'checkbox') el.checked = !!value;
			else el.value = value === undefined || value === null ? '' : String(value);
		};
		Object.keys(EXAMPLE).forEach(function (k) {
			set(k, EXAMPLE[k]);
		});
	}

	document.getElementById('btn-load-example') && document.getElementById('btn-load-example').addEventListener('click', setExample);

	function collectForm() {
		var payload = {
			id: (document.getElementById('id') && document.getElementById('id').value) ? document.getElementById('id').value.trim() : '',
			'execute-command': (document.getElementById('execute-command') && document.getElementById('execute-command').value) ? document.getElementById('execute-command').value.trim() : '',
			'command-working-directory': (document.getElementById('command-working-directory') && document.getElementById('command-working-directory').value) ? document.getElementById('command-working-directory').value.trim() : '',
			'response-message': (document.getElementById('response-message') && document.getElementById('response-message').value) ? document.getElementById('response-message').value.trim() : '',
			'webhook_base_url': (document.getElementById('webhook_base_url') && document.getElementById('webhook_base_url').value) ? document.getElementById('webhook_base_url').value.trim() : '',
			'http-methods': (document.getElementById('http-methods') && document.getElementById('http-methods').value) ? document.getElementById('http-methods').value.trim() : '',
			'success-http-response-code': 200,
			'include-command-output-in-response': (document.getElementById('include-command-output-in-response') && document.getElementById('include-command-output-in-response').checked) || false,
			'response-headers': (document.getElementById('response-headers') && document.getElementById('response-headers').value) ? document.getElementById('response-headers').value.trim() : '',
			'pass-arguments-to-command': (document.getElementById('pass-arguments-to-command') && document.getElementById('pass-arguments-to-command').value) ? document.getElementById('pass-arguments-to-command').value.trim() : '',
			'pass-environment-to-command': (document.getElementById('pass-environment-to-command') && document.getElementById('pass-environment-to-command').value) ? document.getElementById('pass-environment-to-command').value.trim() : '',
			'trigger-rule': (document.getElementById('trigger-rule') && document.getElementById('trigger-rule').value) ? document.getElementById('trigger-rule').value.trim() : '',
			'incoming-payload-content-type': (document.getElementById('incoming-payload-content-type') && document.getElementById('incoming-payload-content-type').value) ? document.getElementById('incoming-payload-content-type').value.trim() : ''
		};
		var codeEl = document.getElementById('success-http-response-code');
		if (codeEl && codeEl.value !== '') payload['success-http-response-code'] = parseInt(codeEl.value, 10) || 200;
		return payload;
	}

	function copyToClipboard(text) {
		if (navigator.clipboard && navigator.clipboard.writeText) {
			return navigator.clipboard.writeText(text);
		}
		var ta = document.createElement('textarea');
		ta.value = text;
		ta.style.position = 'fixed';
		ta.style.left = '-9999px';
		document.body.appendChild(ta);
		ta.select();
		try {
			document.execCommand('copy');
			return Promise.resolve();
		} finally {
			document.body.removeChild(ta);
		}
	}

	function downloadBlob(content, filename, mime) {
		var blob = new Blob([content], { type: mime || 'text/plain' });
		var a = document.createElement('a');
		a.href = URL.createObjectURL(blob);
		a.download = filename;
		a.click();
		URL.revokeObjectURL(a.href);
	}

	document.getElementById('form').onsubmit = function (e) {
		e.preventDefault();
		var I18N = window.I18N;
		if (!I18N) return;
		var lang = getLang();
		var t = I18N[lang] || I18N.zh;
		var payload = collectForm();
		var resultEl;
		if (!payload.id) {
			resultEl = document.getElementById('result');
			resultEl.textContent = t.resultErrorId || 'Please fill in Hook ID.';
			resultEl.className = 'result-area error';
			resultEl.setAttribute('role', 'alert');
			document.getElementById('output').innerHTML = '';
			document.getElementById('actions').innerHTML = '';
			resultEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
			return;
		}
		if (!payload['execute-command']) {
			resultEl = document.getElementById('result');
			resultEl.textContent = t.resultErrorCommand || 'Please fill in execute command.';
			resultEl.className = 'result-area error';
			resultEl.setAttribute('role', 'alert');
			document.getElementById('output').innerHTML = '';
			document.getElementById('actions').innerHTML = '';
			resultEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
			return;
		}
		resultEl = document.getElementById('result');
		var outputEl = document.getElementById('output');
		var actionsEl = document.getElementById('actions');
		var submitBtn = document.getElementById('btn-generate');
		resultEl.textContent = t.generating || 'Generating...';
		resultEl.className = 'result-area';
		resultEl.removeAttribute('role');
		outputEl.innerHTML = '';
		actionsEl.innerHTML = '';
		if (submitBtn) submitBtn.disabled = true;

		fetch('/api/generate', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(payload)
		})
			.then(function (r) {
				return r.text().then(function (text) {
					var data = null;
					try {
						if (text) data = JSON.parse(text);
					} catch (e) { /* ignore */ }
					if (!r.ok) {
						var msg = (data && data.error) ? data.error : (r.statusText || text || 'Request failed');
						throw new Error(msg);
					}
					return data;
				});
			})
			.then(function (data) {
				resultEl.textContent = t.resultSuccess || 'Done. Copy or download:';
				resultEl.className = 'result-area';
				var callUrlLabel = t.outputCallUrl || 'Call URL';
				var curlLabel = t.outputCurl || 'curl';
				var yamlLabel = t.outputYaml || 'YAML';
				var jsonLabel = t.outputJson || 'JSON';
				var html = '';
				if (data.callUrl) {
					html += '<div class="output-block"><label>' + escapeHtml(callUrlLabel) + '</label><pre class="output-pre">' + escapeHtml(data.callUrl) + '</pre></div>';
				}
				if (data.curlExample) {
					html += '<div class="output-block"><label>' + escapeHtml(curlLabel) + '</label><pre class="output-pre">' + escapeHtml(data.curlExample) + '</pre></div>';
				}
				if (data.yaml) {
					html += '<div class="output-block"><details class="output-details" open><summary>' + escapeHtml(yamlLabel) + '</summary><pre class="output-pre output-yaml">' + escapeHtml(data.yaml) + '</pre></details></div>';
				}
				if (data.json) {
					html += '<div class="output-block"><details class="output-details" open><summary>' + escapeHtml(jsonLabel) + '</summary><pre class="output-pre output-json">' + escapeHtml(data.json) + '</pre></details></div>';
				}
				outputEl.innerHTML = html;
				try {
					if (payload.id) localStorage.setItem('webhook-config-ui-id', payload.id);
					if (payload['webhook_base_url']) localStorage.setItem('webhook-config-ui-base-url', payload['webhook_base_url']);
				} catch (e) { /* ignore */ }

				var actions = '';
				if (t.copyAll && (data.yaml || data.curlExample)) {
					actions += '<button type="button" class="pure-button btn-copy" data-copy="all">' + t.copyAll + '</button>';
				}
				if (t.copyCurl && data.curlExample) {
					actions += '<button type="button" class="pure-button btn-copy" data-copy="curl">' + t.copyCurl + '</button>';
				}
				if (t.copyYaml && data.yaml) {
					actions += '<button type="button" class="pure-button btn-copy" data-copy="yaml">' + t.copyYaml + '</button>';
				}
				if (t.copyJson && data.json) {
					actions += '<button type="button" class="pure-button btn-copy" data-copy="json">' + t.copyJson + '</button>';
				}
				if (t.downloadYaml && data.yaml) {
					actions += '<a href="#" class="pure-button btn-download" data-dl="yaml">' + t.downloadYaml + '</a>';
				}
				if (t.downloadJson && data.json) {
					actions += '<a href="#" class="pure-button btn-download" data-dl="json">' + t.downloadJson + '</a>';
				}
				actionsEl.innerHTML = actions;

				var lastData = { callUrl: data.callUrl, curlExample: data.curlExample, yaml: data.yaml, json: data.json };
				function getAllText() {
					var parts = [];
					if (lastData.callUrl) parts.push('# ' + (lang === 'zh' ? '调用 URL' : 'Call URL') + '\n' + lastData.callUrl);
					if (lastData.curlExample) parts.push('# curl\n' + lastData.curlExample);
					if (lastData.yaml) parts.push('# YAML\n' + lastData.yaml);
					if (lastData.json) parts.push('# JSON\n' + lastData.json);
					return parts.join('\n\n');
				}
				actionsEl.querySelectorAll('.btn-copy').forEach(function (btn) {
					btn.addEventListener('click', function () {
						var kind = this.getAttribute('data-copy');
						var text = kind === 'all' ? getAllText() : (kind === 'curl' ? lastData.curlExample : (kind === 'yaml' ? lastData.yaml : lastData.json));
						var origText = kind === 'all' ? t.copyAll : (kind === 'curl' ? t.copyCurl : (kind === 'yaml' ? t.copyYaml : t.copyJson));
						copyToClipboard(text).then(function () {
							btn.textContent = t.copySuccess || 'Copied';
							setTimeout(function () { btn.textContent = origText; }, 1500);
						}).catch(function () {
							btn.textContent = t.copyFailed || 'Copy failed';
							setTimeout(function () { btn.textContent = origText; }, 2000);
						});
					});
				});
				actionsEl.querySelectorAll('.btn-download').forEach(function (a) {
					a.addEventListener('click', function (e) {
						e.preventDefault();
						var kind = this.getAttribute('data-dl');
						var content = kind === 'yaml' ? lastData.yaml : lastData.json;
						downloadBlob(content, kind === 'yaml' ? 'hooks.yaml' : 'hooks.json', kind === 'json' ? 'application/json' : 'application/x-yaml');
					});
				});
				if (outputEl && outputEl.innerHTML) {
					outputEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
				}
			})
			.catch(function (err) {
				resultEl.textContent = (t.requestFailed || 'Request failed: ') + (err && err.message ? err.message : String(err));
				resultEl.className = 'result-area error';
				resultEl.setAttribute('role', 'alert');
				resultEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
			})
			.finally(function () {
				if (submitBtn) submitBtn.disabled = false;
			});
	};

	function escapeHtml(s) {
		if (!s) return '';
		var div = document.createElement('div');
		div.textContent = s;
		return div.innerHTML;
	}
})();
