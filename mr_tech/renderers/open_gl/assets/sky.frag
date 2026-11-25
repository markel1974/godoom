#version 330 core
in vec3 v_ray;
layout (location = 0) out vec4 FragColor;
layout (location = 1) out vec4 BrightColor;

uniform sampler2DArray u_sky[3];
uniform float u_skyLayer;
uniform float u_scrollU;
uniform float u_scrollV;

uniform int u_hasSkyTex;
uniform int u_sunEnabled;
uniform vec3 u_sunDirWorld;
uniform vec4 u_sunColorIntensity;
uniform vec3 u_sunSkyColor;

uniform int u_fogEnabled;
uniform vec3 u_fogColor;
uniform vec3 u_fogSunColor;
uniform float u_fogScatter;

vec4 getSky(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_sky[0], vec3(tc.xy, l));
    if (b == 1) return texture(u_sky[1], vec3(tc.xy, l));
    return texture(u_sky[2], vec3(tc.xy, l));
}

void main() {
    vec3 d = normalize(v_ray);
    const float PI = 3.14159265359;

    float sunAlign = 0.0;
    if (u_sunEnabled == 1) {
        vec3 toSun = normalize(-u_sunDirWorld);
        sunAlign = clamp(dot(d, toSun), 0.0, 1.0);
    }

    vec3 horizonFogColor = u_fogColor;
    if (u_sunEnabled == 1 && u_fogScatter > 0.001) {
        float scatter = clamp(pow(sunAlign, 8.0) * u_fogScatter, 0.0, 1.0);
        horizonFogColor = mix(u_fogColor, u_fogSunColor, scatter);
    }

    vec3 skyRgb;
    float alpha = 1.0;

    if (u_hasSkyTex == 1) {
        float u = atan(d.z, d.x) / (2.0 * PI) + 0.5 + u_scrollU;
        float v = asin(d.y) / PI + 0.5 + u_scrollV;
        vec4 skyCol = getSky(vec3(u, v, u_skyLayer));
        skyRgb = skyCol.rgb;
        alpha = skyCol.a;
        if (u_fogEnabled == 1) {
            float horizonBand = smoothstep(0.24, -0.02, d.y);
            skyRgb = mix(skyRgb, horizonFogColor, horizonBand);
        }
    } else {
        vec3 baseHorizon = (u_fogEnabled == 1) ? horizonFogColor : mix(u_sunSkyColor, vec3(0.76, 0.86, 0.96), 0.58);
        vec3 zenith = max(u_sunSkyColor * 0.45, vec3(0.08, 0.18, 0.42));
        float elev = clamp(max(d.y, 0.0), 0.0, 1.0);
        float gradT = pow(elev, 0.52);
        skyRgb = mix(baseHorizon, zenith, gradT);

        if (u_sunEnabled == 1) {
            float sunHalo = pow(sunAlign, 12.0) * 0.28 + pow(sunAlign, 96.0) * 0.85;
            float sunDisc = smoothstep(0.9991, 0.9996, sunAlign) * 3.2;
            vec3 sunTint = max(u_sunColorIntensity.rgb, vec3(0.0));
            skyRgb += sunTint * ((sunHalo + sunDisc) * max(u_sunColorIntensity.w * 0.45, 0.5));
        }
        if (d.y < 0.0) {
            float belowMix = smoothstep(0.0, -0.25, d.y);
            skyRgb = mix(baseHorizon, baseHorizon * 0.85, belowMix);
        }
        if (u_fogEnabled == 1) {
            float horizonLock = smoothstep(0.06, -0.02, d.y);
            skyRgb = mix(skyRgb, horizonFogColor, horizonLock);
        }
    }

    FragColor = vec4(skyRgb, alpha);
    float lum = dot(skyRgb, vec3(0.2126, 0.7152, 0.0722));
    BrightColor = vec4(lum > 2.4 ? (skyRgb - 2.4) : vec3(0.0), 1.0);
}