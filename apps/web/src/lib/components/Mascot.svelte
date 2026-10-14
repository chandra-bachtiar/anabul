<script lang="ts">
	interface Props {
		mood?: 'senang' | 'khawatir' | 'tidur';
		size?: number;
		alt?: string;
		class?: string;
	}

	let { mood = 'senang', size = 64, alt, class: className = '' }: Props = $props();

	const sumber: Record<string, string> = {
		senang: '/mascot-senang.svg',
		khawatir: '/mascot-khawatir.svg',
		tidur: '/mascot-tidur.svg'
	};

	const label: Record<string, string> = {
		senang: 'Maskot ANABUL sedang senang',
		khawatir: 'Maskot ANABUL khawatir karena budget menipis',
		tidur: 'Maskot ANABUL tidur'
	};
</script>

<img
	src={sumber[mood] ?? sumber.senang}
	width={size}
	height={size}
	alt={alt ?? label[mood] ?? ''}
	class="shrink-0 select-none {className}"
/>
