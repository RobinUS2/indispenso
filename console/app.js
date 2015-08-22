var app = {
	token : function() {
		return localStorage['token'];
	},

	apiErr : function(resp) {
		alert(resp.error);
	},

	showPage : function(name) {
		var currentPage = $('.page-visible');
		var currentPageName = currentPage.attr('data-name');
		currentPage.removeClass('page-visible');
		$('.page[data-name="' + name + '"]').addClass('page-visible');

		// Call load and unload
		if (typeof app.pages[currentPageName]['unload'] === 'function') {
			app.pages[currentPageName]['unload']();
		}
		if (typeof app.pages[name]['load'] === 'function') {
			app.pages[name]['load']();
		}
	},

	run : function() {
		/** Login */
		if (app.token() !== null && app.token().length > 0) {
			app.showPage('home');
		} else {
			app.showPage('login');
		}
	},

	pages : {
		home : {
			load : function() {
				
			}
		},

		login : {
			load : function() {
				$('form#login').submit(function() {
					$.post('/auth', $(this).serialize(), function(resp) {
						if (resp.status === 'OK') {
							localStorage['token'] = resp.session_token;
							app.showPage('home');
						} else {
							app.apiErr(resp);
						}
					}, 'json');
					return false;
				});
			}
		}
	}
};
$(document).ready(function() {
	app.run();
});