var app = {
	token : function() {
		return localStorage['token'];
	},

	username : function() {
		return localStorage['username'];
	},

	apiErr : function(resp) {
		if (resp.error.indexOf('authorized') !== -1) {
			app.logout();
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

	pageInstance : function() {
		return $('.page-visible');
	},

	bindData : function(k, v) {
		$('[data-bind="' + k + '"]', app.pageInstance()).html(v);
	},

	logout : function() {
		delete localStorage['token'];
		delete localStorage['username'];
		app.showPage('login');
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

		profile : {
			load : function() {
				$('form#change-password').submit(function() {
					app.ajax('/user/password', { method: 'PUT', data : $(this).serialize() }).done(function(resp) {
						var resp = app.handleResponse(resp);
						if (resp.status === 'OK') {
							app.logout();
						}
					}, 'json');
					return false;
				});
			},
			unload : function() {
				$('form#change-password').unbind('submit');
			}
		},

		clients : {
			load : function() {
				app.ajax('/clients').done(function(resp) {
					var resp = app.handleResponse(resp);
					var rows = [];
					var listTags = [];
					$(resp.clients).each(function(i, client) {
						var tags = [];
						$(client.Tags).each(function(j, tag) {
							tags.push('<span class="label label-primary">' + tag + '</span>');
							if (listTags.indexOf(tag) === -1) {
								listTags.push(tag);
							}
						});
						rows.push('<tr class="client"><td>' + client.ClientId + '</td><td>' + tags.join("\n") + '</td><td>' + client.LastPing + '</td></tr>');
					});
					app.bindData('clients', rows.join("\n"));

					// List of tags
					var listTagsHtml = [];
					$(listTags).each(function(i, tag) {
						listTagsHtml.push('<li><span class="label label-primary filter-tag clickable" data-included="1" data-tag="' + tag + '">' + tag + '</span></li>');
					});
					app.bindData('tags', listTagsHtml.join("\n"));

					// Filter based on tags
					$('span.filter-tag', app.pageInstance()).click(function() {
						var included = $(this).attr('data-included') === '1';
						if (included) {
							$(this).removeClass('label-primary').addClass('label-default').attr('data-included', '0');
						} else {
							$(this).removeClass('label-default').addClass('label-primary').attr('data-included', '1');
						}

						// On tags
						var onTags = [];
						$('span.filter-tag[data-included="1"]').each(function(i, on) {
							onTags.push($(on).attr('data-tag'));
						});

						// Update list
						$('tr.client').each(function(i, tr) {
							var found = false;
							var trHtml = $(tr).html();
							$(onTags).each(function(j, tag) {
								if (trHtml.indexOf(tag) !== -1) {
									found = true;
									return false;
								}
							});
							if (!found) {
								$(tr).hide();
							} else {
								$(tr).show();
							}
						});

						return false;
					});

					// @todo double click on filter tag only turns on that specific one
				});
			}
		},

		'404' : {
			load : function() {
			}
		},

		templates : {
			load : function() {
			}
		},

		'create-template' : {
			load : function() {
				$('.select2', app.pageInstance()).select2();
			}
		},

		logout : {
			load : function() {
				app.logout();
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