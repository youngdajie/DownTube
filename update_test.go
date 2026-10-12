package main

import "testing"

// TestCompareVersions 验证版本号比较:数字段逐个比较、长度不齐补零、容忍前缀与后缀。
func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.4", "1.2.3", 1},
		{"1.2.3", "1.2.4", -1},
		{"1.10.0", "1.9.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.2", "1.2.0", 0},
		{"1.2.0", "1.2.1", -1},
		{"v1.3.0", "1.2.9", 1},
		{"V1.3.0", "v1.3.0", 0},
		{"1.2.4-beta", "1.2.4", 0},
		{"1.2.4-beta.1", "1.2.4-beta.2", 0}, // 后缀不参与比较
		{"1.2.4.1", "1.2.4", 1},
		{"0.2.0", "1.0.0", -1},
		{"dev", "1.0.0", -1},
		{"1.0.0", "dev", 1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// TestAllowedUpdateHost 验证下载地址白名单:只放行官方发布渠道的 https 地址。
func TestAllowedUpdateHost(t *testing.T) {
	allowed := []string{
		"https://cnb.cool/youngdajie/downtube/-/releases/download/V1.2.4/DownTube-Setup-1.2.4.exe",
		"https://asset.cnb.cool/assets/xxx",
		"https://github.com/youngdajie/DownTube/releases/download/V1.2.4/DownTube-Setup-1.2.4.exe",
		"https://objects.githubusercontent.com/xxx",
	}
	for _, u := range allowed {
		if !allowedUpdateHost(u) {
			t.Errorf("allowedUpdateHost(%q) = false, want true", u)
		}
	}
	blocked := []string{
		"http://cnb.cool/youngdajie/downtube/-/releases/download/V1.2.4/x.exe", // 非 https
		"https://evil.com/DownTube-Setup-1.2.4.exe",
		"https://cnb.cool.evil.com/x.exe",
		"https://fakegithub.com/x.exe",
		"ftp://cnb.cool/x.exe",
		"",
		"not a url",
	}
	for _, u := range blocked {
		if allowedUpdateHost(u) {
			t.Errorf("allowedUpdateHost(%q) = true, want false", u)
		}
	}
}
