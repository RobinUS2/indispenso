var app = {
	token : function() {
		return localStorage['token'];
	},

	username : function() {
		return localStorage['username'];
	},

	userId : function() {
		return localStorage['user_id'];
	},

	apiErr : function(resp) {
		if (resp.error.indexOf('authorized') !== -1) {
			app.logout();
		}
		app.alert('warning', 'Error', resp.error);
	},

	// Only to be used for fast visual hiding of elements, real validation is on the server side
	userRoles : function() {
		var l = localStorage['user_roles'];
		if (typeof l === 'undefined' || l === null) {
			return [];
		}
		return l.split(',');
	},

	alert : function(type, title, message) {
		$('#alert').html('<div class="alert alert-' + type + '" role="alert"><strong>' + title + '</strong> ' + message + '</div>');
		setTimeout(function() {
			$('#alert > div').slideUp();
		}, 2000);
	},

	_params : {},
	getParam : function(k) {
		var v = app._params[k];
		if (typeof v === 'undefined') {
			return null;
		}
		return v;
	},

	showPage : function(input) {
		// Params?
		var qp = input.indexOf('?');
		var name = input;
		if (qp !== -1) {
			name = input.substr(0, qp);
			var paramStr = input.substr(qp + 1);
			var paramParts = paramStr.split('&');
			$(paramParts).each(function(i, part) {
				var kv = part.split('=');
				app._params[kv[0]] = decodeURIComponent(kv[1]);
			});
		}

		// Unload
		if (typeof app.pages[currentPageName] !== 'undefined' && typeof app.pages[currentPageName]['unload'] === 'function') {
			app.pages[currentPageName]['unload']();
		}

		// New page
		history.pushState(null, null, '#!' + input);
		var currentPage = $('.page-visible');
		var currentPageName = currentPage.attr('data-name');
		currentPage.removeClass('page-visible');
		$('.page[data-name="' + name + '"]').addClass('page-visible');

		// 404?
		if (typeof app.pages[name] === 'undefined' && name !== '404') {
			this.showPage('404');
			return;
		}

		// Load
		if (typeof app.pages[name]['load'] === 'function') {
			app.pages[name]['load']();
		}

		// Hide elements that are not visible to your role
		app.updateRolesDom();
	},

	updateRolesDom : function() {
		if (app.userRoles().length === 0) {
			return;
		}
		$('[data-roles]').each(function(i, elm) {
			var roles = $(elm).attr('data-roles').split(',');
			var hasAll = true;
			$(roles).each(function(i, role) {
				if (app.userRoles().indexOf(role) === -1) {
					hasAll = false;
				}
			});
			if (!hasAll) {
				$(this).hide();
			} else {
				if ($(elm).hasClass('page')) {
					if ($(elm).hasClass('page-visible')) {
						$(this).show();
					} else {
						$(this).hide();
					}
				}
			}
		});
	},

	initNav : function() {
		$('a[data-nav]').unbind('click');
		$('a[data-nav]').click(function() {
			app.showPage($(this).attr('data-nav'));
			// Hide nav on mobile
			if ($('button.navbar-toggle').is(':visible') && $('.navbar-collapse').hasClass('in')) {
				$('button.navbar-toggle').click();
			}
			return false;
		});
	},

	pollWork : function() {
		setInterval(function() {
			if (typeof app.token() !== 'undefined' && app.token() !== null && app.token().length > 0) {
				app.ajax('/consensus/pending').done(function(resp) {
					var resp = app.handleResponse(resp);
					if (resp.status === 'OK') {
						if (Object.keys(resp.work).length > 0) {
							// New work, notify
							if (!app._openNotification) {
								app.showDesktopNotification('Pending Approval', '', 'pending');
							}
						}
					}
				});
			}
		}, 3000);
	},

	run : function() {
		/** Top menu */
		app.initNav();

		/** Check for work notifications */
		this.pollWork();

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
		delete localStorage['user_id'];
		delete localStorage['user_roles'];
		app.showPage('login');
	},

	_openNotification : false,

	showDesktopNotification : function(title, msg, targetPage) {
		if (!Notification) {
			return
		}
		var notification = new Notification(title, {
	      body: msg,
	    });
	    app._openNotification = true;
	    notification.onclick = function () {
	     	window.focus();
	     	app._openNotification = false;
	     	if (targetPage !== 'undefined' && targetPage !== null) {
	      		app.showPage(targetPage);
	    	}
	    };
	},

	requestDesktopNotification : function() {
		if (!Notification) {
			return
		}
		if (Notification.permission !== 'granted') {
		    Notification.requestPermission();
		}
	},

	pages : {
		home : {
			load : function() {
				app.ajax('/clients').done(function(resp) {
					var resp = app.handleResponse(resp);
					if (resp.status === 'OK') {
						app.bindData('number-of-clients', resp.clients.length);
					}
				});
				app.ajax('/consensus/pending').done(function(resp) {
					var resp = app.handleResponse(resp);
					if (resp.status === 'OK') {
						app.bindData('number-of-pending', Object.keys(resp.requests).length);
						app.bindData('number-of-work', Object.keys(resp.work).length);
					}
				});

				// Ask notification permissions
				app.requestDesktopNotification();
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
						var lastTime = client.LastPing.substr(0, client.LastPing.indexOf('.')).replace('T', ' ');
						rows.push('<tr class="client"><td>' + client.ClientId + '</td><td>' + tags.join("\n") + '</td><td>' + lastTime + '</td></tr>');
					});
					app.bindData('clients', rows.join("\n"));
					$('table', app.pageInstance()).DataTable();

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

					// Double click on filter tag only turns on that specific one
					$('.filter-tag').dblclick(function() { 
						var tag = $(this).attr('data-tag'); 
						$('.filter-tag:not([data-tag="' + tag + '"])').click(); 
						return false; 
					});
				});
			}
		},

		'404' : {
			load : function() {
			}
		},

		pending : {
			load : function() {
				app.ajax('/templates').done(function(resp) {
					var resp = app.handleResponse(resp);
					var templates = resp.templates;

					app.ajax('/users/names').done(function(resp) {
						var resp = app.handleResponse(resp);
						var userNames = resp.users;
						var userMap = {};
						$(userNames).each(function(i, user) {
							userMap[user.Id] = user;
						});

						app.ajax('/consensus/pending').done(function(resp) {
							var resp = app.handleResponse(resp);
							if (resp.status === 'OK') {
								var workKeys = Object.keys(resp.work);
								var workHtml = [];
								$(workKeys).each(function(i, workKey) {
									var work = resp.work[workKey];
									var template = templates[work.TemplateId];
									var user = userMap[work.RequestUserId];
									if (typeof user === 'undefined') {
										user = {
											Id : '',
											Username : ''
										}
									}

									var lines = [];
									lines.push('<tr>');
									lines.push('<td>' + template.Title + '</td>');
									lines.push('<td>' + user.Username + '</td>');
									lines.push('<td><div class="btn-group btn-group-xs pull-right"><span class="btn btn-success approve-request" data-roles="approver" data-id="' + work.Id + '">Approve</span> <span class="btn btn-default cancel-request" data-id="' + work.Id + '">Cancel</span></div></td>');
									lines.push('</tr>');
									workHtml.push(lines.join(''));
								});
								app.bindData('work', workHtml.join("\n"));
								$('.approve-request', app.pageInstance()).click(function() {
									var id = $(this).attr('data-id');
									app.ajax('/consensus/approve', { method: 'POST', data : { id : id } }).done(function(resp) {
										var resp = app.handleResponse(resp);
										if (resp.status === 'OK') {
											app.showPage('pending');
										}
									});
								});

								var workHtml = [];
								var requestKeys = Object.keys(resp.requests);
								$(requestKeys).each(function(i, requestKey) {
									var request = resp.requests[requestKey];
									var template = templates[request.TemplateId];
									var user = userMap[request.RequestUserId];
									if (typeof user === 'undefined') {
										user = {
											Id : '',
											Username : ''
										}
									}

									var lines = [];
									lines.push('<tr>');
									lines.push('<td>' + template.Title + '</td>');
									lines.push('<td>' + user.Username + '</td>');
									lines.push('<td>');
									if (user.Id === app.userId() || app.userRoles().indexOf('admin') !== -1) {
										lines.push('<div class="btn-group btn-group-xs pull-right"><span class="btn btn-default cancel-request" data-id="' + request.Id + '">Cancel</span></div>');
									}
									lines.push('</td>');
									lines.push('</tr>');
									workHtml.push(lines.join(''));
								});
								app.bindData('pending', workHtml.join("\n"));
								$('table', app.pageInstance()).DataTable();
								$('.cancel-request', app.pageInstance()).click(function() {
									var id = $(this).attr('data-id');
									app.ajax('/consensus/request?id=' + id, { method: 'DELETE' }).done(function(resp) {
										var resp = app.handleResponse(resp);
										if (resp.status === 'OK') {
											app.showPage('pending');
										}
									});
								});
							}
						});
					});
				});
			}
		},

		users : {
			load : function() {
				app.ajax('/users').done(function(resp) {
					var resp = app.handleResponse(resp);
					var html = [];
					for (var k in resp.users) {
						if (!resp.users.hasOwnProperty(k)) {
							continue;
						}
						var obj = resp.users[k];
						var lines = [];
						lines.push('<tr>');
						lines.push('<td>' + obj.Username + '</td>');
						lines.push('<td>' + Object.keys(obj.Roles).join(', ') + '</td>');
						lines.push('<td><div class="btn-group btn-group-xs pull-right"><span class="btn btn-default delete-user" data-username="' + obj.Username + '">Delete</span></div></td>');
						lines.push('</tr>');
						html.push(lines.join("\n"));
					}
					app.bindData('users', html.join("\n"));
					$('table', app.pageInstance()).DataTable();

					$('.delete-user').click(function() {
						var username = $(this).attr('data-username');
						if (!confirm('Are you sure you want to delete "' + username + '"?')) {
							return;
						}
						app.ajax('/user?username=' + username, { method: 'DELETE' }).done(function(resp) {
							var resp = app.handleResponse(resp);
							if (resp.status === 'OK') {
								app.showPage('users');
							}
						});
					});
				});
			},
			unload : function() {
				$('.delete-user').unbind('click');
			}
		},

		'create-user' : {
			load : function() {
				$('form#create-user').submit(function() {
					var d = $(this).serialize();
					try { d['roles'] = $('#roles', app.pageInstance()).val().join(','); } catch (e) {}
					app.ajax('/user', { method: 'POST', data : d }).done(function(resp) {
						var resp = app.handleResponse(resp);
						if (resp.status === 'OK') {
							app.showPage('users');
						}
					}, 'json');
					return false;
				});
			},
			unload : function() {
				$('form#create-user').unbind('submit');
			}
		},

		templates : {
			load : function() {
				app.ajax('/templates').done(function(resp) {
					var resp = app.handleResponse(resp);
					var templatesHtml = [];
					for (var k in resp.templates) {
						if (!resp.templates.hasOwnProperty(k)) {
							continue;
						}
						var template = resp.templates[k];
						var lines = [];
						lines.push('<tr>');
						lines.push('<td>' + template.Title + '</td>');
						var tags = [];
						$(template.Acl.IncludedTags).each(function(i, tag) {
							tags.push('<span class="label label-primary">' + tag + '</span>');
						});
						$(template.Acl.ExcludedTags).each(function(i, tag) {
							tags.push('<span class="label label-danger">' + tag + '</span>');
						});
						if (tags.length === 0) {
							tags.push('<span class="label label-success">ANY</span>');
						}
						lines.push('<td>' + tags.join(" ") + '</td>');
						lines.push('<td><div class="btn-group btn-group-xs pull-right"><a class="btn btn-default" data-nav="request-execution?id=' + template.Id + '" data-roles="requester" href="#">Execute</a> <span class="btn btn-default delete-template" data-roles="admin" data-id="' + template.Id + '">Delete</span></div></td>');
						lines.push('</tr>');
						templatesHtml.push(lines.join("\n"));
					}
					app.bindData('templates', templatesHtml.join("\n"));
					$('table', app.pageInstance()).DataTable();
					app.initNav();
					app.updateRolesDom();
					$('.delete-template').click(function() {
						var id = $(this).attr('data-id');
						if (!confirm('Are you sure you want to delete this template?')) {
							return;
						}
						app.ajax('/template?id=' + id, { method: 'DELETE' }).done(function(resp) {
							var resp = app.handleResponse(resp);
							if (resp.status === 'OK') {
								app.showPage('templates');
							}
						});
					});
				});
			}
		},

		'request-execution' : {
			load : function() {
				$('.request-execution', app.pageInstance()).show();
				$('.select-clients', app.pageInstance()).hide();

				var id = app.getParam('id');
				if (id === null || id.length < 1) {
					console.log('No id');
					return app.showPage('templates');
				}
				app.ajax('/templates').done(function(resp) {
					var resp = app.handleResponse(resp);
					var template = resp.templates[id];
					if (typeof template === 'undefined' || template === null) {
						console.log('Template not found');
						return app.showPage('templates');
					}

					// Title
					app.bindData('template-title', template.Title);
					app.bindData('template-description', template.Description);
					app.bindData('template-command', template.Command);
					app.bindData('template-minAuth', template.Acl.MinAuth);

					// Get eligible clients
					app.ajax('/clients?filter_tags_include=' + encodeURIComponent(template.Acl.IncludedTags.join(',')) + '&filter_tags_exclude=' + encodeURIComponent(template.Acl.ExcludedTags.join(','))).done(function(resp) {
						var resp = app.handleResponse(resp);
						var rows = [];
						$(resp.clients).each(function(i, client) {
							var tags = [];
							$(client.Tags).each(function(j, tag) {
								tags.push('<span class="label label-primary">' + tag + '</span>');
							});
							rows.push('<tr class="client"><td><input type="checkbox" class="select-client" data-id="' + client.ClientId + '" value="1"></td><td>' + client.ClientId + '</td><td>' + tags.join("\n") + '</td><td>' + client.LastPing + '</td></tr>');
						});
						app.bindData('clients', rows.join("\n"));
						$('table', app.pageInstance()).DataTable();

						// Make button active
						$('.request-execution > .btn', app.pageInstance()).unbind('click');
						$('.request-execution > .btn', app.pageInstance()).click(function() {
							$('.request-execution', app.pageInstance()).hide();
							$('.select-clients', app.pageInstance()).show();
						});

						// Toggle all
						$('.toggle-clients', app.pageInstance()).unbind('click');
						$('.toggle-clients', app.pageInstance()).click(function() {
							var on = $(this).attr('data-state') === '1';
							if (on) {
								// Turn off
								$('.select-client', app.pageInstance()).prop("checked", false);
								$(this).attr('data-state', '0');
							} else {
								// Turn ON
								$('.select-client', app.pageInstance()).prop("checked", true);
								$(this).attr('data-state', '1');
							}
						});

						// Execute
						$('.do-request', app.pageInstance()).unbind('click');
						$('.do-request', app.pageInstance()).click(function() {
							// Confirm
							if (!confirm('Are you sure you want to continue?')) {
								return false;
							}

							// List clients
							var clientIds = [];
							$('.select-client:checked').each(function(i, cb) {
								clientIds.push($(cb).attr('data-id'));
							});
							if (clientIds.length < 1) {
								app.alert('warning', 'No clients', 'You need to select at least one target client');
								return;
							}

							// Request
							app.ajax('/consensus/request', { method: 'POST', data : { template : template.Id, clients : clientIds.join(',') } }).done(function(resp) {
								var resp = app.handleResponse(resp);
								if (resp.status === 'OK') {
									app.showPage('pending');
								}
							});

							return false;
						});
					});
				});
			}
		},

		'create-template' : {
			load : function() {
				app.ajax('/tags').done(function(resp) {
					var resp = app.handleResponse(resp);
					var tagOptions = [];
					$(resp.tags).each(function(i, tag) {
						tagOptions.push('<option value="' + tag + '">' + tag + '</option>');
					});
					app.bindData('tags', tagOptions.join("\n"));
					$('.select2', app.pageInstance()).select2();
				});

				$('form#create-template').submit(function() {
					var d = $(this).serialize();
					try { d['includedTags'] = $('#includedTags', app.pageInstance()).val().join(','); } catch (e) {}
					try { d['excludedTags'] = $('#excludedTags', app.pageInstance()).val().join(','); } catch (e) {}
					app.ajax('/template', { method: 'POST', data : d }).done(function(resp) {
						var resp = app.handleResponse(resp);
						if (resp.status === 'OK') {
							app.showPage('templates');
						}
					}, 'json');
					return false;
				});
			},
			unload : function() {
				$('form#create-template').unbind('submit');
			}
		},

		logs : {
			load : function() {
				var id = app.getParam('id');
				var client = app.getParam('client');
				app.ajax('/client/' + client + '/cmd/' + id + '/logs').done(function(resp) { 
					var resp = app.handleResponse(resp);
					if (resp.status !== 'OK') {
						app.showPage('history');
						return;
					}

					var lis = [];
					$(resp.log_output).each(function(i, line) {
						lis.push(line);
					});
					app.bindData('out', lis.join("\n"));

					var lis = [];
					$(resp.log_error).each(function(i, line) {
						lis.push(line);
					});
					app.bindData('err', lis.join("\n"));
				});
			}
		},

		history : {
			load : function() {
				app.ajax('/templates').done(function(resp) { 
					var resp = app.handleResponse(resp);
					var templates = resp.templates;
					app.ajax('/dispatched').done(function(resp) { 
						var dispatched = resp.dispatched;

						// Print template
						var html = [];
						$(dispatched).each(function(i, elm) {
							var template = templates[elm.TemplateId];
							html.push('<tr><td>' + template.Title + '</td><td>' + elm.Id + '</td><td>' + elm.State + '</td><td><div class="btn-group btn-group-xs pull-right"><a class="btn btn-default" data-nav="logs?id=' + elm.Id + '&client=' + elm.ClientId + '" href="#">Logs</a></div></td></tr>');
						});
						app.bindData('dispatched', html.join("\n"));
						$('table', app.pageInstance()).DataTable();
						app.initNav(); // Bind logs button
					});
				});
			}
		},

		logout : {
			load : function() {
				app.logout();
			}
		},

		login : {
			load : function() {
				$('.navbar-nav').hide();
				$('form#login').submit(function() {
					$.post('/auth', $(this).serialize(), function(resp) {
						if (resp.status === 'OK') {
							localStorage['token'] = resp.session_token;
							localStorage['user_id'] = resp.user_id;
							localStorage['username'] = $('form#login input[name="username"]').val();
							localStorage['user_roles'] = resp.user_roles.join(',');
							app.alert('info', 'Login successful', 'Welcome back ' + localStorage['username']);
							$('.navbar-nav').show();
							app.showPage('home');
						} else {
							app.apiErr(resp);
						}
					}, 'json');
					return false;
				});
			},
			unload : function() {
				$('.navbar-nav').show();
				$('form#login').unbind('submit');
			}
		}
	}
};
$(document).ready(function() {
	app.run();
});