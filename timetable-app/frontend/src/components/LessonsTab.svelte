<script>
        export let classes = [];
        export let lessons = [];
        export let curClass = 0;
        export let l;
        export let subjects = [];
        export let teachers = [];
        export let subjName;
        export let updateLesson;
        export let removeLesson;
        export let addLessonForClass;
        export let downloadRefsJSON;
        export let importRefsJSON;
</script>

<section class="card">
        <div class="card-head">
                <h2>Учебный план (уроки)</h2>
                <div class="csvbar">
                        <button class="mini" on:click={() => downloadRefsJSON('lessons')}>⬇ JSON</button>
                        <label class="mini file">⬆<input type="file" accept=".json" on:change={(e) => importRefsJSON('lessons', e)} /></label>
                </div>
        </div>
        <div class="lesson-form">
                <select bind:value={curClass}><option value={0}>Выберите класс…</option>{#each classes as x}<option value={x.id}>{x.name}</option>{/each}</select>
        </div>
        {#if curClass}
                {@const cl = lessons.filter((x) => x.class_id === curClass)}
                <table class="data">
                        <thead><tr><th>Предмет</th><th>Учитель</th><th>Ч/нед</th><th></th></tr></thead>
                        <tbody>
                                {#each cl as x}
                                        <tr>
                                                <td>{subjName(subjects, x.subject_id)}</td>
                                                <td><select bind:value={x.teacher_id} on:change={() => updateLesson(x)}>{#each teachers as t}<option value={t.id}>{t.name}</option>{/each}</select></td>
                                                <td><input type="number" min="1" max="40" bind:value={x.hours_per_week} on:change={() => updateLesson(x)} /></td>
                                                <td class="act"><button class="danger sm" on:click={() => removeLesson(x.id)}>✕</button></td>
                                        </tr>
                                {/each}
                                <tr class="addrow">
                                        <td><select bind:value={l.subject_id}><option value={0}>Предмет</option>{#each subjects as s}<option value={s.id}>{s.name}</option>{/each}</select></td>
                                        <td><select bind:value={l.teacher_id}><option value={0}>Учитель</option>{#each teachers as t}<option value={t.id}>{t.name}</option>{/each}</select></td>
                                        <td><input type="number" min="1" max="40" bind:value={l.hours_per_week} placeholder="Ч/нед" /></td>
                                        <td class="act"><button class="primary sm" on:click={addLessonForClass}>+ Добавить</button></td>
                                </tr>
                        </tbody>
                </table>
                {#if cl.length === 0}<p class="muted">У этого класса пока нет уроков. Добавьте предмет и учителя выше.</p>{/if}
        {:else}
                <p class="muted">Выберите класс, чтобы увидеть и редактировать его учебный план (предмет + учитель + часы в неделю).</p>
        {/if}
</section>
