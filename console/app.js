var app = {
	token : function() {
		return localStorage['token'];
	},

	apiErr : function(resp) {
		alert(resp.error);
	},

	showPage : function(name) {
		$('.page-visible').removeClass('page-visible');
		$('.page[data-name="' + name + '"]').addClass('page-visible');
	},

	run : function() {
		/** Login */
		if (app.token() !== null && app.token().length > 0) {
			app.showPage('home');
		} else {
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
};
$(document).ready(function() {
	app.run();
});