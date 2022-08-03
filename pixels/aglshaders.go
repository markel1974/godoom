package pixels

var baseCanvasVertexShader = `
#version 330 core

in vec2  aPosition;
in vec4  aColor;
in vec2  aTexCoords;
in float aIntensity;
in vec4  aClipRect;
in float aIsClipped;

out vec4  vColor;
out vec2  vTexCoords;
out float vIntensity;
out vec2  vPosition;
out vec4  vClipRect;

uniform mat3 uTransform;
uniform vec4 uBounds;

void main() {
	vec2 transPos = (uTransform * vec3(aPosition, 1.0)).xy;
	vec2 normPos = (transPos - uBounds.xy) / uBounds.zw * 2 - vec2(1, 1);
	gl_Position = vec4(normPos, 0.0, 1.0);

	vColor = aColor;
	vPosition = aPosition;
	vTexCoords = aTexCoords;
	vIntensity = aIntensity;
	vClipRect = aClipRect;
}
`

var baseCanvasFragmentShader = `
#version 330 core

in vec4  vColor;
in vec2  vTexCoords;
in float vIntensity;
in vec4  vClipRect;

out vec4 fragColor;

uniform vec4 uColorMask;
uniform vec4 uTexBounds;
uniform sampler2D uTexture;

void main() {
	if ((vClipRect != vec4(0,0,0,0)) && (gl_FragCoord.x < vClipRect.x || gl_FragCoord.y < vClipRect.y || gl_FragCoord.x > vClipRect.z || gl_FragCoord.y > vClipRect.w)) {
		discard;
	}

	if (vIntensity == 0) {
		fragColor = uColorMask * vColor;
	} else {
		fragColor = vec4(0, 0, 0, 0);
		fragColor += (1 - vIntensity) * vColor;
		vec2 t = (vTexCoords - uTexBounds.xy) / uTexBounds.zw;
		fragColor += vIntensity * vColor * texture(uTexture, t);
		fragColor *= uColorMask;
	}
}
`
