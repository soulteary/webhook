/**
 * Webhook config generator UI.
 * Expects window.I18N from server-rendered page (config/page.yaml).
 */
(function () {
	'use strict';

	// Ensure I18N is always an object (template may output JSON string in some envs)
	if (typeof window.I18N === 'string') {
		try {
			window.I18N = JSON.parse(window.I18N);
		} catch (e) {
			window.I18N = {};
		}
	}
	if (!window.I18N || typeof window.I18N !== 'object') {
		window.I18N = {};
	}

	var LANG_STORAGE_KEY = 'webhook-config-ui-lang';

	function getLang() {
		return localStorage.getItem(LANG_STORAGE_KEY) || 'zh';
	}

	function applyLang(lang) {
		var I18N = window.I18N;
		if (!I18N || typeof I18N !== 'object') return;
		var langMap = I18N[lang];
		if (!langMap || typeof langMap !== 'object') lang = 'zh';
		if (!I18N[lang]) lang = Object.keys(I18N)[0] || 'zh';
		localStorage.setItem(LANG_STORAGE_KEY, lang);
		document.documentElement.lang = lang === 'zh' ? 'zh-CN' : 'en';
		document.title = (I18N[lang] && I18N[lang].title) || document.title;
		document.querySelectorAll('[data-i18n]').forEach(function (el) {
			var key = el.getAttribute('data-i18n');
			if (key && I18N[lang] && I18N[lang][key] !== undefined) el.textContent = I18N[lang][key];
		});
		document.querySelectorAll('.lang-switch a').forEach(function (a) {
			a.classList.toggle('active', a.getAttribute('data-lang') === lang);
		});
	}

	var langSwitchLinks = document.querySelectorAll('.lang-switch a');
	langSwitchLinks.forEach(function (a) {
		a.addEventListener('click', function (e) {
			e.preventDefault();
			e.stopPropagation();
			var lang = (e.currentTarget && e.currentTarget.getAttribute('data-lang')) || getLang();
			applyLang(lang);
			return false;
		});
	});
	try {
		applyLang(getLang());
	} catch (e) { /* ensure rest of script runs */ }

	var JSON_TEMPLATES = {
		'response-headers': '[{"name":"X-Custom","value":"ok"}]',
		'pass-arguments-to-command': '[{"source":"payload","name":"repo"}]',
		'pass-environment-to-command': '[{"source":"payload","envname":"REPO","name":"repo"}]',
		'trigger-rule': '{"match":{"type":"value","parameter":{"source":"header","name":"X-Key"},"value":"secret"}}'
	};
	function injectJsonTemplateButtons() {
		var I18N = window.I18N;
		var lang = getLang();
		var t = (I18N && I18N[lang]) ? I18N[lang] : {};
		var label = t.btnInsertExample || 'Insert example';
		Object.keys(JSON_TEMPLATES).forEach(function (id) {
			var el = document.getElementById(id);
			if (!el || el.tagName !== 'TEXTAREA') return;
			var wrap = el.closest('.config-item');
			if (!wrap || wrap.querySelector('.btn-insert-json')) return;
			var btn = document.createElement('button');
			btn.type = 'button';
			btn.className = 'pure-button btn-insert-json';
			btn.setAttribute('data-i18n', 'btnInsertExample');
			btn.setAttribute('data-target', id);
			btn.textContent = label;
			btn.addEventListener('click', function () {
				var targetId = this.getAttribute('data-target');
				var target = document.getElementById(targetId);
				if (target && JSON_TEMPLATES[targetId]) target.value = JSON_TEMPLATES[targetId];
			});
			el.parentNode.insertBefore(btn, el);
		});
	}
	injectJsonTemplateButtons();

	var saveToDirEnabled = false;
	fetch('api/capabilities').then(function (r) { return r.ok ? r.json() : {}; }).then(function (d) {
		saveToDirEnabled = !!(d && d.saveToDir);
	}).catch(function () { /* ignore */ });

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

	var btnLoadExample = document.getElementById('btn-load-example');
	if (btnLoadExample) {
		btnLoadExample.addEventListener('click', function (e) {
			e.preventDefault();
			setExample();
		});
	}

	function isValidJson(s) {
		if (!s || typeof s !== 'string' || !s.trim()) return { valid: true };
		try {
			JSON.parse(s.trim());
			return { valid: true };
		} catch (e) {
			return { valid: false, error: e.message };
		}
	}
	function isValidBaseUrl(s) {
		if (!s || typeof s !== 'string' || !s.trim()) return true;
		var t = s.trim();
		return t.indexOf('http://') === 0 || t.indexOf('https://') === 0;
	}
	function isValidStatusCode(n) {
		var num = parseInt(n, 10);
		return !isNaN(num) && num >= 100 && num <= 999;
	}

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

	var formEl = document.getElementById('form');
	if (!formEl) formEl = document.querySelector('form');
	formEl && (formEl.onsubmit = function (e) {
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
		var jsonFields = [
			{ id: 'response-headers', key: 'responseHeadersLabel' },
			{ id: 'pass-arguments-to-command', key: 'passArgumentsLabel' },
			{ id: 'pass-environment-to-command', key: 'passEnvironmentLabel' },
			{ id: 'trigger-rule', key: 'triggerRuleLabel' }
		];
		for (var i = 0; i < jsonFields.length; i++) {
			var el = document.getElementById(jsonFields[i].id);
			if (!el || !el.value || !el.value.trim()) continue;
			var res = isValidJson(el.value);
			if (!res.valid) {
				resultEl = document.getElementById('result');
				var label = (t[jsonFields[i].key] || jsonFields[i].id) + ': ';
				resultEl.textContent = label + (t.validationJsonInvalid || 'Must be valid JSON.');
				resultEl.className = 'result-area error';
				resultEl.setAttribute('role', 'alert');
				document.getElementById('output').innerHTML = '';
				document.getElementById('actions').innerHTML = '';
				resultEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
				return;
			}
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

		var apiUrl = (function () {
			var base = document.querySelector('base[href]');
			var baseUrl = base ? base.getAttribute('href') : '';
			if (baseUrl) baseUrl = baseUrl.replace(/\/$/, '');
			return baseUrl ? baseUrl + '/api/generate' : 'api/generate';
		})();
		fetch(apiUrl, {
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
						var msg;
						if (r.status === 413) {
							msg = { type: 'bodyTooLarge', text: t.errorRequestBodyTooLarge || 'Request too large.' };
						} else if (data && data.error) {
							msg = { type: 'serverValidation', text: (t.errorServerValidation || 'Validation failed: ') + data.error };
						} else {
							msg = { type: 'requestFailed', text: (t.requestFailed || 'Request failed: ') + (r.statusText || text || 'Unknown error') };
						}
						throw msg;
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
				html += '<div class="next-steps" role="region" aria-labelledby="next-steps-title"><p class="next-steps-title" id="next-steps-title">' + escapeHtml(t.nextStepsTitle || 'Next steps') + '</p><ul class="next-steps-list"><li>' + escapeHtml(t.nextStepsCopy || 'Copy the generated config into your hooks file.') + '</li><li>' + escapeHtml(t.nextStepsHooksDir || 'With -hooks-dir you can save directly here.') + '</li><li>' + escapeHtml(t.nextStepsUrlPrefix || 'Ensure -urlprefix matches the call URL.') + '</li></ul></div>';
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
				if (saveToDirEnabled && (data.yaml || data.json)) {
					var saveLabel = t.saveToDirLabel || (lang === 'zh' ? '保存到目录' : 'Save to directory');
					var saveBtnLabel = t.saveBtnLabel || (lang === 'zh' ? '保存' : 'Save');
					var defaultName = (payload.id || 'hook').replace(/[^a-zA-Z0-9_-]/g, '-') + '.yaml';
					actions += '<div class="save-to-dir-block"><label>' + escapeHtml(saveLabel) + '</label> <input type="text" id="save-filename" class="pure-input save-filename-input" value="' + escapeHtml(defaultName) + '" aria-describedby="save-msg"> <select id="save-format" class="save-format-select" aria-label="Format"><option value="yaml">YAML</option><option value="json">JSON</option></select> <button type="button" class="pure-button btn-save-to-dir" aria-live="polite">' + escapeHtml(saveBtnLabel) + '</button> <span id="save-msg" class="save-msg" role="status" aria-live="polite"></span></div>';
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
				var btnSave = actionsEl.querySelector('.btn-save-to-dir');
				if (btnSave) {
					var saveMsg = document.getElementById('save-msg');
					btnSave.addEventListener('click', function () {
						var filenameEl = document.getElementById('save-filename');
						var formatEl = document.getElementById('save-format');
						var filename = (filenameEl && filenameEl.value) ? filenameEl.value.trim() : '';
						var format = (formatEl && formatEl.value) || 'yaml';
						var content = format === 'json' ? lastData.json : lastData.yaml;
						if (!filename || !content) {
							if (saveMsg) { saveMsg.textContent = t.saveHintGenerateFirst || (lang === 'zh' ? '请先生成配置并填写文件名' : 'Generate first and enter filename'); saveMsg.style.color = ''; }
							return;
						}
						btnSave.disabled = true;
						if (saveMsg) { saveMsg.textContent = t.saveSaving || (lang === 'zh' ? '保存中…' : 'Saving…'); saveMsg.style.color = ''; }
						fetch('api/save', {
							method: 'POST',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({ filename: filename, content: content })
						}).then(function (r) {
							return r.json().then(function (body) {
								if (!r.ok) throw new Error(body.error || r.statusText);
								if (saveMsg) { saveMsg.textContent = (t.saveSuccess || (lang === 'zh' ? '已保存: ' : 'Saved: ')) + (body.ok || filename); saveMsg.style.color = ''; }
							});
						}).catch(function (err) {
							if (saveMsg) { saveMsg.textContent = (err && err.message) || (t.saveFailed || 'Save failed'); saveMsg.style.color = '#c00'; }
						}).finally(function () {
							btnSave.disabled = false;
						});
					});
				}
				if (outputEl && outputEl.innerHTML) {
					outputEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
				}
			})
			.catch(function (err) {
				var msg;
				if (err && typeof err === 'object' && err.type === 'bodyTooLarge') {
					msg = err.text;
				} else if (err && typeof err === 'object' && err.type === 'serverValidation') {
					msg = err.text;
				} else if (err && typeof err === 'object' && err.type === 'requestFailed') {
					msg = err.text;
				} else if (err && (err.message === 'Failed to fetch' || (err.name === 'TypeError' && err.message && err.message.indexOf('fetch') !== -1))) {
					msg = t.errorNetwork || 'Network error. Check connection and retry.';
				} else {
					msg = (t.requestFailed || 'Request failed: ') + (err && err.message ? err.message : String(err));
				}
				resultEl.textContent = msg;
				resultEl.className = 'result-area error';
				resultEl.setAttribute('role', 'alert');
				resultEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
			})
			.finally(function () {
				if (submitBtn) submitBtn.disabled = false;
			});
	});

	function escapeHtml(s) {
		if (!s) return '';
		var div = document.createElement('div');
		div.textContent = s;
		return div.innerHTML;
	}
})();
