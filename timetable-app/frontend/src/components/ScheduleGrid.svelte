<script>
        export let vm;
</script>

{#if vm.overviewMode}
        <div class="overview">
                {#each vm.grid as row (row.id)}
                        <div class="mini" class:is-sub={vm.isSubRow(row.id)}>
                                <button class="mini-title" on:click={() => vm.focusRow(row.id)} title="Открыть этот класс отдельно">
                                        {row.label}{#if vm.subParentName(row.id)}<span class="sub-badge">подгруппа {vm.subParentName(row.id)}</span>{/if}
                                </button>
                                <table>
                                        <thead><tr><th class="d"></th>{#each Array(vm.slots) as _, si}<th>{#if vm.periodLabel(si)}{vm.periodLabel(si)}{:else}П{si + 1}{/if}</th>{/each}</tr></thead>
                                        <tbody>
                                                {#each row.cells as dayCells, di}
                                                        <tr>
                                                                <td class="day">{vm.dayName(vm.activeDayIdx[di])}</td>
                                                                {#each dayCells as cell, si}
                                                                        <td
                                                                                class:filled={!!cell}
                                                                                class:selected={vm.selectedEntry && cell && vm.selectedEntry.id === cell.id}
                                                                                class:drop-target={vm.dropTarget && vm.dropTarget.rowId === row.id && vm.dropTarget.day === vm.activeDayIdx[di] && vm.dropTarget.slot === si}
                                                                                title={cell ? cell.label : ""}
                                                                                data-cell
                                                                                data-day={vm.activeDayIdx[di]}
                                                                                data-slot={si}
                                                                                data-row={row.id}
                                                                                data-kind={vm.kind}
                                                                                on:pointerdown={(e) => vm.onPointerDown(e, cell, vm.kind, row.id, vm.activeDayIdx[di], si)}
                                                                        >
                                                                                {#if cell}
                                                                                        <div class="chip" class:half={vm.cellSubs("class", row.id, vm.activeDayIdx[di], si).length > 0} class:conflict={cell.conflict} style="background:{cell.conflict ? '#dc2626' : vm.subjectColor(cell.subject_id)}">
                                                                                                {vm.subjShort(vm.subjects, cell.subject_id)}
                                                                                        </div>
                                                                                {/if}
                                                                                {#each vm.cellSubs("class", row.id, vm.activeDayIdx[di], si) as sc}
                                                                                        <div class="chip half" style="background:{vm.subjectColor(sc.subject_id)}" title="Подгруппа {vm.classes.find((c) => c.id === sc.class_id)?.name || ''}">
                                                                                                {vm.subjShort(vm.subjects, sc.subject_id)}
                                                                                        </div>
                                                                                {/each}
                                                                        </td>
                                                                {/each}
                                                        </tr>
                                                {/each}
                                        </tbody>
                                </table>
                        </div>
                {/each}
        </div>
{:else}
        <div class="grid-scroll">
                {#each vm.grid as row (row.id)}
                        <div class="class-block" class:compact={vm.compact} class:is-sub={vm.isSubRow(row.id)}>
                                <h3>{row.label} {#if vm.subParentName(row.id)}<span class="sub-badge">подгруппа {vm.subParentName(row.id)}</span>{/if}</h3>
                                <table class:compact={vm.compact}>
                                        <thead><tr><th class="day-h">День</th>{#each Array(vm.slots) as _, si}<th>П{si + 1}{#if vm.periodLabel(si)}<span class="tm">{vm.periodLabel(si)}</span>{/if}</th>{/each}</tr></thead>
                                        <tbody>
                                                {#each row.cells as dayCells, di}
                                                        <tr>
                                                                <td class="day">{vm.dayName(vm.activeDayIdx[di])}</td>
                                                                {#each dayCells as cell, si}
                                                                        <td
                                                                                class:filled={!!cell}
                                                                                class:selected={vm.selectedEntry && cell && vm.selectedEntry.id === cell.id}
                                                                                class:drop-target={vm.dropTarget && vm.dropTarget.rowId === row.id && vm.dropTarget.day === vm.activeDayIdx[di] && vm.dropTarget.slot === si}
                                                                                title={cell ? cell.label : ""}
                                                                                data-cell
                                                                                data-day={vm.activeDayIdx[di]}
                                                                                data-slot={si}
                                                                                data-row={row.id}
                                                                                data-kind={vm.kind}
                                                                                on:pointerdown={(e) => vm.onPointerDown(e, cell, vm.kind, row.id, vm.activeDayIdx[di], si)}
                                                                        >
                                                                                {#if cell}
                                                                                        <div class="chip" class:half={vm.cellSubs(vm.kind, row.id, vm.activeDayIdx[di], si).length > 0} class:conflict={cell.conflict} style="background:{cell.conflict ? '#dc2626' : vm.subjectColor(cell.subject_id)}">
                                                                                                <b>{vm.subjName(vm.subjects, cell.subject_id)}</b>
                                                                                                {#if vm.pdfShowTeacher}<span>{vm.teachName(vm.teachers, cell.teacher_id)}</span>{/if}
                                                                                                {#if vm.pdfShowRoom && cell.room_id}<span>{vm.roomById.get(cell.room_id)?.name || ""}</span>{/if}
                                                                                        </div>
                                                                                        <button class="cell-x" title="Удалить" on:click={(e) => { e.stopPropagation(); vm.removeEntry(cell.id); }}>✕</button>
                                                                                {/if}
                                                                                {#each vm.cellSubs(vm.kind, row.id, vm.activeDayIdx[di], si) as sc}
                                                                                        <div
                                                                                                class="chip half"
                                                                                                role="button"
                                                                                                tabindex="-1"
                                                                                                class:conflict={vm.conflictIDs.has(sc.id)}
                                                                                                style="background:{vm.conflictIDs.has(sc.id) ? '#dc2626' : vm.subjectColor(sc.subject_id)}"
                                                                                                title="Подгруппа {vm.classById.get(sc.class_id)?.name || ''}"
                                                                                                on:pointerdown={(e) => { e.stopPropagation(); vm.onPointerDown(e, vm.cellAt("class", sc.class_id, vm.activeDayIdx[di], si), "class", row.id, vm.activeDayIdx[di], si); }}
                                                                                        >
                                                                                                <b>{vm.subjName(vm.subjects, sc.subject_id)}</b>
                                                                                                {#if vm.pdfShowTeacher}<span>{vm.teachName(vm.teachers, sc.teacher_id)}</span>{/if}
                                                                                        </div>
                                                                                {/each}
                                                                        </td>
                                                                {/each}
                                                        </tr>
                                                {/each}
                                        </tbody>
                                </table>
                        </div>
                {/each}
        </div>
{/if}

{#if vm.ghost}
        <div class="drag-ghost" style="left:{vm.ghost.x}px; top:{vm.ghost.y}px;">{vm.ghost.label}</div>
{/if}
