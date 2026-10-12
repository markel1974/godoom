#version 330 core
layout (location = 0) out vec4 gPositionDepth;
layout (location = 1) out vec4 gNormal;
layout (location = 2) out vec4 gAlbedoSpec;
layout (location = 3) out vec4 gEmissive;

in vec3 ViewPos;
in vec3 TexCoords;
in float FragDepth;
in float IsFullbright;

uniform sampler2DArray u_texture[4];
uniform sampler2DArray u_normalMap[4];
uniform sampler2DArray u_emissiveMap[4];
uniform float u_emissiveIntensity;

vec3 getNormal(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_normalMap[0], vec3(tc.xy, l)).rgb;
    if (b == 1) return texture(u_normalMap[1], vec3(tc.xy, l)).rgb;
    if (b == 2) return texture(u_normalMap[2], vec3(tc.xy, l)).rgb;
    return texture(u_normalMap[3], vec3(tc.xy, l)).rgb;
}

vec4 getDiffuse(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_texture[0], vec3(tc.xy, l));
    if (b == 1) return texture(u_texture[1], vec3(tc.xy, l));
    if (b == 2) return texture(u_texture[2], vec3(tc.xy, l));
    return texture(u_texture[3], vec3(tc.xy, l));
}

vec3 getEmissive(vec3 tc) {
    int b = int(tc.z) / 1000;
    float l = mod(tc.z, 1000.0);
    if (b == 0) return texture(u_emissiveMap[0], vec3(tc.xy, l)).rgb;
    if (b == 1) return texture(u_emissiveMap[1], vec3(tc.xy, l)).rgb;
    if (b == 2) return texture(u_emissiveMap[2], vec3(tc.xy, l)).rgb;
    return texture(u_emissiveMap[3], vec3(tc.xy, l)).rgb;
}

void main() {
    vec4 texColor = getDiffuse(TexCoords);
    if(texColor.a < 0.5) discard;

    // Scrive Posizione e Profondità
    gPositionDepth = vec4(ViewPos, FragDepth);

    // Calcoliamo la variazione di ViewPos rispetto alle coordinate schermo X e Y
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);

    // Il prodotto vettoriale delle derivate produce una normale geometrica coerente
    vec3 geoNormal = normalize(cross(dp1, dp2));
    if (dot(geoNormal, ViewPos) > 0.0) {
        geoNormal = -geoNormal;
    }

    // Lettura della Normal Map
    vec3 mapColor = getNormal(TexCoords);
    vec3 finalNormal = geoNormal;

    if (length(mapColor) > 0.1) {
        vec3 unpacked = (mapColor * 2.0) - 1.0;
        vec3 mapNormal = normalize(unpacked);

        vec2 duv1 = dFdx(TexCoords.xy);
        vec2 duv2 = dFdy(TexCoords.xy);

        vec3 dp2perp = cross(dp2, geoNormal);
        vec3 dp1perp = cross(geoNormal, dp1);
        vec3 T = dp2perp * duv1.x + dp1perp * duv2.x;
        vec3 B = dp2perp * duv1.y + dp1perp * duv2.y;

        float lenT = length(T);
        float lenB = length(B);
        if (lenT > 1e-8 && lenB > 1e-8) {
            T /= lenT;
            // Gram-Schmidt: T ortogonale alla normale
            T = normalize(T - geoNormal * dot(geoNormal, T));
            // Ricostruzione di B per avere un TBN ortonormale coerente
            B = normalize(cross(geoNormal, T));
            mat3 TBN = mat3(T, B, geoNormal);
            finalNormal = normalize(TBN * mapNormal);
        }
    }

    gNormal = vec4(finalNormal, 1.0);
    gAlbedoSpec = vec4(texColor.rgb, 1.0);

    vec3 emColor = getEmissive(TexCoords) * u_emissiveIntensity;
    gEmissive = vec4(emColor, IsFullbright);
}