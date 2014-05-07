/** App object */
var app = {
	userData: null,
	/** Default error callback for API issues */
	defaultErrCallback : function(xhr, status, err) {
		console.log(xhr, status, err);
		app.alert(err)
	},
	/** Simple alerts */
	alert : function(msg) {
		alert(msg);
	},
	/** API helper */
	api : function (method, data, callback, errCallback) {
    	if (errCallback == null || typeof errCallback !== 'function') {
    		errCallback = app.defaultErrCallback;
    	}
		$.ajax('/api?method=' + encodeURIComponent(method), {
		    'data': JSON.stringify(data), //{action:'x',params:['a','b','c']}
		    'type': 'POST',
		    'contentType': 'application/json',
		    'error': function(xhr, status, err) {
		    	errCallback(xhr, status, err);
		    }
		}).done(function( data ) {
			if (typeof data['error'] !== 'undefined' && data['error'] != null) {
				errCallback(null, 200, data['error']);
			} else {
				callback(data);
			}
		});
	},
	/** Show a screen */
	showScreen : function (id) {
		$('div.content-pane').removeClass('content-visible');
		$('div.content-pane[data-id="' + id + '"]').addClass('content-visible');
		return true;
	},
	/** Json serialize */
	jsonSerialize : function(form) {
		var data = {};
		$(form).find('input, select').each(function(i, elm) {
			var name = $(elm).attr('data-name');
			var val = $(elm).val();
			if (typeof name === 'undefined' || name == null || name.length == 0) {
				return false;
			}
			data[name] = val;
		});
		return data;
	}
};

/** Show panel */


// Example handshake
// jsApi('mirror', {a:1}, function(x){console.log(x);});

/** WEB APPLICATION CODE */

/** Login */
$(document).ready(function() {
	/** Are we logged in? */
	app.api('mirror', {a:1}, function(x) {
		/** Yes */
		app.showScreen('home');
	}, function() {
		/** No */
		app.showScreen('login');
		$('form#login').submit(function() {
			app.api('auth', app.jsonSerialize(this), function(data) {
				app.userData = data.user;
				console.log(app.userData);
				app.showScreen('home');
			});
			return false;
		});
	});
});