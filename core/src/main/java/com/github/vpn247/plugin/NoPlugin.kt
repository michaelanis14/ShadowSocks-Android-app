package com.github.vpn247.plugin

import com.github.vpn247.Core.app

object NoPlugin : Plugin() {
    override val id: String get() = ""
    override val label: CharSequence get() = app.getText(com.github.vpn247.core.R.string.plugin_disabled)
}
