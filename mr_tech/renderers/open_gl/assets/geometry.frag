#version 330 core
layout (location = 0) out vec4 gPositionDepth;
layout (location = 1) out vec4 gNormal;

in vec3 ViewPos;
in vec3 TexCoords;
in float FragDepth;

uniform sampler2DArray u_texture[4];
uniform sampler2DArray u_normalMap[4];

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

void main() {
    // Early discard per la trasparenza
    if(getDiffuse(TexCoords).a < 0.5) discard;

    // Scrive Posizione e Profondità (necessari per SSAO)
    gPositionDepth = vec4(ViewPos, FragDepth);

    // Calcoliamo la variazione di ViewPos rispetto alle coordinate schermo X e Y
    vec3 dp1 = dFdx(ViewPos);
    vec3 dp2 = dFdy(ViewPos);

    // Il prodotto vettoriale delle derivate ci dà la normale geometrica perfetta
    vec3 geoNormal = normalize(cross(dp1, dp2));

    // Sicurezza Anti-Backface (Winding safety)
    // In View Space la telecamera guarda verso -Z, una normale che guarda
    // verso la telecamera deve avere Z positiva.
    if (geoNormal.z < 0.0) {
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

    // Scrive la normale calcolata nel G-Buffer per un SSAO molto più dettagliato
    gNormal = vec4(finalNormal, 1.0);
}