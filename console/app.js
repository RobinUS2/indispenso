var app = {
	token : function() {
		return localStorage['token'];
	},

	username : function() {
		return localStorage['username'];
	},

	apiErr : function(resp) {
		if (resp.error.indexOf('authorized') !== -1) {
			delete localStorage['token'];
			delete localStorage['username'];
			app.showPage('login');
		}
		alert(resp.error);
	},

	showPage : function(name) {
		history.pushState(null, null, '#!' + name);
		var currentPage = $('.page-visible');
		var currentPageName = currentPage.attr('data-name');
		currentPage.removeClass('page-visible');
		$('.page[data-name="' + name + '"]').addClass('page-visible');

		// Call load and unload
		if (typeof app.pages[currentPageName] !== 'undefined' && typeof app.pages[currentPageName]['unload'] === 'function') {
			app.pages[currentPageName]['unload']();
		}
		if (typeof app.pages[name] === 'undefined' && name !== '404') {
			this.showPage('404');
			return;
		}
		if (typeof app.pages[name]['load'] === 'function') {
			app.pages[name]['load']();
		}
	},

	run : function() {
		/** Top menu */
		$('a[data-nav]').click(function() {
			app.showPage($(this).attr('data-nav'));
			return false;
		});

		/** Init route based of location */
		var h = document.location.hash.substr(2);
		if (h.length > 0) {
			app.showPage(h);
			return;
		}

		/** Login */
		if (typeof app.token() !== 'undefined' && app.token() !== null && app.token().length > 0) {
			app.showPage('home');
		} else {
			app.showPage('login');
		}
	},

	ajax : function(url, opts) {
		if (typeof opts === 'undefined' || opts === null) {
			opts = {};
		}
		if (typeof opts["headers"] === 'undefined') {
			opts["headers"] = {};
		}
		opts["headers"]["X-Auth-User"] = app.username();
		opts["headers"]["X-Auth-Session"] = app.token();
		opts["dataType"] = 'json';
		var x = $.ajax(url, opts);
		return x;
	},

	handleResponse : function(resp) {
		if (resp['status'] !== 'OK') {
			app.apiErr(resp);
		}
		return resp;
	},

	bindData : function(k, v) {
		$('[data-bind="' + k + '"]', '.page-visible').html(v);
	},

	pages : {
		home : {
			load : function() {
				app.ajax('/clients').done(function(resp) {
					var resp = app.handleResponse(resp);
					app.bindData('number-of-clients', resp.clients.length);
				});
			}
		},

		clients : {
			load : function() {
				app.ajax('/clients').done(function(resp) {
					var resp = app.handleResponse(resp);
					var rows = [];
					$(resp.clients).each(function(i, client) {
						var tags = [];
						$(client.Tags).each(function(j, tag) {
							tags.push('<span class="label label-default">' + tag + '</span>');
						});
						rows.push('<tr><td>' + client.ClientId + '</td><td>' + tags.join("\n") + '</td><td>' + client.LastPing + '</td></tr>');
					});
					app.bindData('clients', rows.join("\n"));
				});
			}
		},

		'404' : {
			load : function() {
			}
		},

		login : {
			load : function() {
				$('form#login').submit(function() {
					$.post('/auth', $(this).serialize(), function(resp) {
						if (resp.status === 'OK') {
							localStorage['token'] = resp.session_token;
							localStorage['username'] = $('form#login input[name="username"]').val();
							app.showPage('home');
						} else {
							app.apiErr(resp);
						}
					}, 'json');
					return false;
				});
			},
			unload : function() {
				$('form#login').unbind('submit');
			}
		}
	}
};
$(document).ready(function() {
	app.run();
});