package com.github.vpn247.aidl;

import com.github.vpn247.aidl.IShadowsocksServiceCallback;

interface IShadowsocksService {
  int getState();
  String getProfileName();

  void registerCallback(in IShadowsocksServiceCallback cb);
  void startListeningForBandwidth(in IShadowsocksServiceCallback cb);
  oneway void stopListeningForBandwidth(in IShadowsocksServiceCallback cb);
  oneway void unregisterCallback(in IShadowsocksServiceCallback cb);
}
